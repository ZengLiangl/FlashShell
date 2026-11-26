package cmds

import (
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
)

// MyBatis 批量插入日志解析：
//
//	==>  Preparing: INSERT INTO example_table (id,...,amount,...,currency,...) VALUES (?,?...), (?,?...)
//	==> Parameters: 1000000000000000001(Long), 1(Long), ..., 4271(BigDecimal), ..., USD(String), ...
//
// Preparing 行只给出列名和行数（全是占位符），真正的值在紧随其后的 Parameters 行里，
// 两者靠同一 traceId + 同一个 Mapper 关联。这里把 Parameters 行按列名还原成结构化记录，
// 再按「金额 + 币种」（可叠加交易类型等维度）汇总。

// logStatsSQLBatch 一次批量插入（一个 Preparing + 一个 Parameters）
type logStatsSQLBatch struct {
	Table     string
	Columns   []string
	Rows      [][]string // 每行的原始值（null 用空串表示）
	RowCount  int
	Truncated bool // Parameters 行疑似被日志采集截断
	TraceID   string
	Thread    string
	Timestamp string
}

// logStatsAggGroup 一个汇总分组
type logStatsAggGroup struct {
	TradeType string
	Currency  string
	Count     int
	Amount    *big.Rat
	AmountRaw string // 便于原样打印的十进制字符串
}

// logStatsAggResult 汇总结果
type logStatsAggResult struct {
	Table   string
	Batches int
	Rows    int
	// DeclaredRows 各批次 Preparing 声明的总行数（真实应插入行数）
	DeclaredRows int
	// TruncatedBatches 被日志截断的批次数
	TruncatedBatches int
	// SkippedTruncated 是否跳过了截断批次
	SkippedTruncated bool
	Truncated        bool
	Groups           []logStatsAggGroup
	TotalByCur       map[string]*logStatsAggGroup
	TotalCount       int
	TotalAmount      *big.Rat
	Columns          []string
	AmountCol        string
	CurrencyCol      string
	TradeCol         string
}

var (
	// Preparing: INSERT INTO <table> (col1,col2,...) VALUES
	reSQLInsertPreparing = regexp.MustCompile(`(?is)Preparing:\s*INSERT\s+INTO\s+([A-Za-z0-9_` + "`" + `.]+)\s*\(([^)]*)\)\s*VALUES`)
	// Parameters: 一段值
	reSQLParameters = regexp.MustCompile(`(?s)==>\s*Parameters:\s*(.*)$`)
	// 值后面的类型后缀：(String) (Long) (BigDecimal) …
	reSQLValueType = regexp.MustCompile(`\((String|Long|Integer|Short|Byte|Double|Float|BigDecimal|BigInteger|Boolean|Timestamp|Date|Time|LocalDateTime|LocalDate|LocalTime|byte\[\])\)`)
	// 日志行前缀：时间 traceId [thread]
	reLogLinePrefix = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d{1,6})?)\s+(\d+)?\s*\[([^\]]*)\]`)
)

// parseSQLParameters 把 "1(Long), 2(String), null, 3.50(BigDecimal)" 拆成值列表。
// null 没有类型后缀，需要单独识别；值里可能含逗号和括号（如带格式的金额），
// 所以以「(类型)」为分隔而不是按逗号切。
func parseSQLParameters(text string) []string {
	vals := make([]string, 0, 64)
	n := len(text)
	pos := 0
	for pos < n {
		for pos < n && (text[pos] == ' ' || text[pos] == ',' || text[pos] == '\n' || text[pos] == '\r' || text[pos] == '\t') {
			pos++
		}
		if pos >= n {
			break
		}
		if strings.HasPrefix(text[pos:], "null") {
			after := pos + 4
			if after >= n || text[after] == ',' || text[after] == ' ' || text[after] == '\n' || text[after] == '\r' {
				vals = append(vals, "")
				pos = after
				continue
			}
		}
		loc := reSQLValueType.FindStringSubmatchIndex(text[pos:])
		if loc == nil {
			// 末尾可能被截断，把剩余内容当作最后一个值
			rest := strings.TrimSpace(text[pos:])
			if rest != "" {
				vals = append(vals, strings.Trim(rest, ","))
			}
			break
		}
		raw := strings.TrimSpace(text[pos : pos+loc[0]])
		raw = strings.TrimSuffix(raw, ",")
		raw = strings.TrimSpace(raw)
		if raw == "null" {
			raw = ""
		}
		vals = append(vals, raw)
		pos += loc[1]
	}
	return vals
}

// parseSQLInsertPreparing 解析列名与行数
func parseSQLInsertPreparing(line string) (table string, cols []string, rowCount int, ok bool) {
	m := reSQLInsertPreparing.FindStringSubmatch(line)
	if m == nil {
		return "", nil, 0, false
	}
	table = m[1]
	for _, c := range strings.Split(m[2], ",") {
		c = strings.TrimSpace(strings.Trim(c, "`"))
		if c != "" {
			cols = append(cols, strings.ToLower(c))
		}
	}
	if len(cols) == 0 {
		return "", nil, 0, false
	}
	// VALUES 后面每组 "(?,?,...)" 是一行
	valuesPart := line[len(m[0]):]
	rowCount = strings.Count(valuesPart, "(?")
	if rowCount == 0 {
		rowCount = 1
	}
	return table, cols, rowCount, true
}

// buildSQLBatches 把 Preparing 与紧随其后的 Parameters 配成批次。
//
// 一个 traceId 下会有多个批次（同一线程循环插入），所以不能只按 traceId 配；
// 用「同 traceId + 同线程 + Parameters 时间戳不早于 Preparing」的最近一条来配。
func buildSQLBatches(hits []logStatsHit, table string) ([]logStatsSQLBatch, bool) {
	type pending struct {
		table string
		cols  []string
		rows  int
		ts    string
		trace string
		thead string
		used  bool
	}

	// 按日志正文自带的时间戳（毫秒精度）排序。
	// 不能用 @timestamp：它只精确到秒，同一秒内的多条 Preparing/Parameters 会乱序。
	sorted := make([]logStatsHit, len(hits))
	copy(sorted, hits)
	lineTS := func(h logStatsHit) string {
		if m := reLogLinePrefix.FindStringSubmatch(h.Raw); m != nil && m[1] != "" {
			return m[1]
		}
		return h.TS
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := lineTS(sorted[i]), lineTS(sorted[j])
		if a != b {
			return a < b
		}
		// 同一毫秒内 Preparing 必须排在 Parameters 之前
		ai := strings.Contains(sorted[i].Raw, "Preparing:")
		bi := strings.Contains(sorted[j].Raw, "Preparing:")
		if ai != bi {
			return ai
		}
		return false
	})

	var pendings []*pending
	var batches []logStatsSQLBatch
	anyTruncated := false

	for _, h := range sorted {
		line := h.Raw
		ts, trace, thread := h.TS, h.Trace, ""
		if m := reLogLinePrefix.FindStringSubmatch(line); m != nil {
			if m[1] != "" {
				ts = m[1]
			}
			if m[2] != "" {
				trace = m[2]
			}
			thread = m[3]
		}

		if strings.Contains(line, "Preparing:") {
			tbl, cols, rows, ok := parseSQLInsertPreparing(line)
			if !ok {
				continue
			}
			if table != "" && !strings.EqualFold(tbl, table) {
				continue
			}
			pendings = append(pendings, &pending{table: tbl, cols: cols, rows: rows, ts: ts, trace: trace, thead: thread})
			continue
		}

		if !strings.Contains(line, "Parameters:") {
			continue
		}
		m := reSQLParameters.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// 找同 trace（或同线程）且尚未使用的最近一条 Preparing
		var p *pending
		for i := len(pendings) - 1; i >= 0; i-- {
			cand := pendings[i]
			if cand.used {
				continue
			}
			if trace != "" && cand.trace != "" && cand.trace != trace {
				continue
			}
			if trace == "" && thread != "" && cand.thead != "" && cand.thead != thread {
				continue
			}
			p = cand
			break
		}
		if p == nil {
			continue
		}
		p.used = true

		vals := parseSQLParameters(m[1])
		if len(vals) == 0 {
			continue
		}
		ncol := len(p.cols)
		full := len(vals) / ncol
		rows := make([][]string, 0, full)
		for i := 0; i < full; i++ {
			rows = append(rows, vals[i*ncol:(i+1)*ncol])
		}
		truncated := len(vals)%ncol != 0 || (p.rows > 0 && full < p.rows)
		if truncated {
			anyTruncated = true
		}
		batches = append(batches, logStatsSQLBatch{
			Table:     p.table,
			Columns:   p.cols,
			Rows:      rows,
			RowCount:  p.rows,
			Truncated: truncated,
			TraceID:   p.trace,
			Thread:    p.thead,
			Timestamp: p.ts,
		})
	}
	return batches, anyTruncated
}

// aggregateSQLBatches 按 金额 + 币种（可选叠加交易类型）汇总
func aggregateSQLBatches(batches []logStatsSQLBatch, amountCol, currencyCol, tradeCol string) *logStatsAggResult {
	return aggregateSQLBatchesOpt(batches, amountCol, currencyCol, tradeCol, false)
}

// aggregateSQLBatchesOpt skipTruncated=true 时只统计完整的批次，
// 避免用「被截断的部分数据」冒充完整合计。
func aggregateSQLBatchesOpt(batches []logStatsSQLBatch, amountCol, currencyCol, tradeCol string, skipTruncated bool) *logStatsAggResult {
	res := &logStatsAggResult{
		Table:       "",
		Batches:     len(batches),
		TotalByCur:  make(map[string]*logStatsAggGroup),
		TotalAmount: new(big.Rat),
	}
	type key struct{ trade, currency string }
	groups := map[key]*logStatsAggGroup{}
	order := make([]key, 0, 16)

	for _, b := range batches {
		if res.Table == "" {
			res.Table = b.Table
		}
		res.Columns = b.Columns
		res.DeclaredRows += b.RowCount
		if b.Truncated {
			res.Truncated = true
			res.TruncatedBatches++
			if skipTruncated {
				res.SkippedTruncated = true
				continue
			}
		}
		idxAmount := indexOfColumn(b.Columns, amountCol)
		idxCurrency := indexOfColumn(b.Columns, currencyCol)
		idxTrade := -1
		if tradeCol != "" {
			idxTrade = indexOfColumn(b.Columns, tradeCol)
		}
		if idxAmount < 0 {
			continue
		}
		for _, row := range b.Rows {
			if idxAmount >= len(row) {
				continue
			}
			res.Rows++
			amt := new(big.Rat)
			if s := strings.TrimSpace(row[idxAmount]); s != "" {
				if r, ok := new(big.Rat).SetString(s); ok {
					amt = r
				}
			}
			cur := ""
			if idxCurrency >= 0 && idxCurrency < len(row) {
				cur = strings.TrimSpace(row[idxCurrency])
			}
			trade := ""
			if idxTrade >= 0 && idxTrade < len(row) {
				trade = strings.TrimSpace(row[idxTrade])
			}
			k := key{trade: trade, currency: cur}
			g := groups[k]
			if g == nil {
				g = &logStatsAggGroup{TradeType: trade, Currency: cur, Amount: new(big.Rat)}
				groups[k] = g
				order = append(order, k)
			}
			g.Count++
			g.Amount.Add(g.Amount, amt)
			res.TotalAmount.Add(res.TotalAmount, amt)
			res.TotalCount++

			tg := res.TotalByCur[cur]
			if tg == nil {
				tg = &logStatsAggGroup{Currency: cur, Amount: new(big.Rat)}
				res.TotalByCur[cur] = tg
			}
			tg.Count++
			tg.Amount.Add(tg.Amount, amt)
		}
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].currency != order[j].currency {
			return order[i].currency < order[j].currency
		}
		return order[i].trade < order[j].trade
	})
	for _, k := range order {
		g := groups[k]
		g.AmountRaw = formatRat(g.Amount)
		res.Groups = append(res.Groups, *g)
	}
	return res
}

func indexOfColumn(cols []string, name string) int {
	if name == "" {
		return -1
	}
	for i, c := range cols {
		if strings.EqualFold(c, name) {
			return i
		}
	}
	return -1
}

// formatRat 把有理数转成十进制字符串（保留小数位，去掉多余的 0）
func formatRat(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	s := r.FloatString(4)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// formatSQLAgg 生成汇总输出
func formatSQLAgg(r *logStatsAggResult, tradeCol string) []string {
	if r == nil {
		return []string{"SQL 汇总失败：空结果"}
	}
	lines := []string{
		fmt.Sprintf("表      : %s", r.Table),
		fmt.Sprintf("批次    : %d 个批量 INSERT；声明 %d 行，实际解析 %d 行", r.Batches, r.DeclaredRows, r.Rows),
	}
	if r.Truncated {
		pct := 0.0
		if r.DeclaredRows > 0 {
			pct = float64(r.Rows) / float64(r.DeclaredRows) * 100
		}
		lines = append(lines, fmt.Sprintf(
			"注意    : %d 个批次的 Parameters 行被日志采集截断（单行 512KB 上限），仅覆盖 %.1f%% 的明细%s",
			r.TruncatedBatches, pct,
			map[bool]string{true: "；已跳过截断批次，合计只含完整批次", false: "，合计会偏小"}[r.SkippedTruncated]))
	}
	lines = append(lines, "")

	if len(r.Groups) == 0 {
		lines = append(lines, "（没有解析到可汇总的明细）")
		return lines
	}

	header := fmt.Sprintf("%-12s %-10s %8s %18s", "币种", "交易类型", "条数", "金额")
	if tradeCol == "" {
		header = fmt.Sprintf("%-12s %8s %18s", "币种", "条数", "金额")
	}
	lines = append(lines, "── 按币种"+map[bool]string{true: " / 交易类型", false: ""}[tradeCol != ""]+" 汇总 ──")
	lines = append(lines, header)
	for _, g := range r.Groups {
		if tradeCol == "" {
			lines = append(lines, fmt.Sprintf("%-12s %8d %18s", g.Currency, g.Count, g.AmountRaw))
		} else {
			lines = append(lines, fmt.Sprintf("%-12s %-10s %8d %18s", g.Currency, g.TradeType, g.Count, g.AmountRaw))
		}
	}

	// 币种合计
	curs := make([]string, 0, len(r.TotalByCur))
	for c := range r.TotalByCur {
		curs = append(curs, c)
	}
	sort.Strings(curs)
	lines = append(lines, "", "── 币种合计 ──")
	for _, c := range curs {
		g := r.TotalByCur[c]
		g.AmountRaw = formatRat(g.Amount)
		lines = append(lines, fmt.Sprintf("%-12s %8d %18s", c, g.Count, g.AmountRaw))
	}
	lines = append(lines, fmt.Sprintf("%-12s %8d %18s", "全部", r.TotalCount, formatRat(r.TotalAmount)))
	return lines
}

// 供命令层使用的小工具
func normalizeSQLTable(s string) string {
	return strings.ToLower(strings.TrimSpace(strings.Trim(s, "`")))
}

// sqlAmountColumns 常见金额列候选
var sqlAmountColumnCandidates = []string{"amount", "settlement_amount", "trade_amount"}
var sqlCurrencyColumnCandidates = []string{"currency", "settlement_currency"}
var sqlTradeColumnCandidates = []string{"trade_type"}

func pickColumn(cols []string, preferred []string, fallback string) string {
	for _, p := range preferred {
		if indexOfColumn(cols, p) >= 0 {
			return p
		}
	}
	return fallback
}

// collectSQLTraceIDs 从 Preparing 行里收集 traceId（去重、保持顺序）
func collectSQLTraceIDs(hits []logStatsHit) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		tid := h.Trace
		if tid == "" {
			if m := reLogLinePrefix.FindStringSubmatch(h.Raw); m != nil {
				tid = m[2]
			}
		}
		if tid == "" || seen[tid] {
			continue
		}
		seen[tid] = true
		out = append(out, tid)
	}
	return out
}
