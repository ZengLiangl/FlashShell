package cmds

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"FlashDock/define"
	"FlashDock/utils"
)

// log-stats：从 Kibana / Elasticsearch 拉取一段时间内的日志，按正则提取数值并汇总求和。
//
// 典型用法（把 Kibana Discover 的地址整段粘进来，时间范围/查询/索引自动识别）：
//
//	log-stats "http://kibana.example.com:5601/app/discover#/?_g=(time:(from:now%2Fw,to:now%2Fw))&_a=(index:'00000000-0000-0000-0000-000000000000',query:(language:kuery,query:%22some+metric%22))"
//
// 命中的日志行形如：
//
//	2026-01-01 00:00:00.000 000000000000 [worker-1] INFO  com.example.Service - [handler,144] - 统计数量：123
//
// 会把所有命中的「统计数量：N」列出来，并汇总求和。
func RegLogStatsCmd() {
	CmdManager.RegSpecialCmd("log-stats", doLogStats)
}

const (
	logStatsDefaultField     = "message"
	logStatsDefaultTimeField = "@timestamp"
	logStatsDefaultPattern   = `统计数量\s*[：:＝=]\s*(-?\d+)`
	// 对比项日志形如：… - 统计数量2:321
	logStatsDefaultComparePattern = `统计数量2\s*[：:＝=]\s*(-?\d+)`
	logStatsDefaultPageSize       = 1000
	logStatsDefaultMaxDocs        = 200000
	logStatsDefaultTimeout        = 120 * time.Second

	logStatsDedupDefault   = "message"
	logStatsDupSampleLimit = 20

	// 默认按天分片查询：ES 按天建索引，分片后每次只碰当天少数分片，
	// 避免一次扫 150 个分片导致 ES 超时并静默返回部分结果
	logStatsDefaultChunk = 24 * time.Hour
)

// logStatsSpec log-stats 的完整参数
type logStatsSpec struct {
	RequestURL string

	BaseURL string // Kibana 根地址（含 basePath）
	ESURL   string // 可选：直连 ES 地址
	Index   string // 索引模式或 data view id

	From     time.Time
	To       time.Time
	FromExpr string
	ToExpr   string

	RangePreset string // --range 原始值（本周 / this-month / 近2月 …）
	RangeLabel  string // 规范化后的展示名

	Query      string
	QueryMode  string // phrase | match | term | query_string
	Field      string
	TimeField  string
	TimeSource string // message（默认，用日志正文时间）| field（用 @timestamp）

	Pattern     *regexp.Regexp
	PatternText string

	// SQLTable 非空时，额外解析该表的 MyBatis 批量 INSERT 日志，按金额/币种汇总
	SQLTable       string
	SQLAmountCol   string
	SQLCurrencyCol string
	SQLTradeCol    string
	// SQLIncludeTruncated 是否把被日志截断的批次也计入（默认跳过，避免用部分数据冒充合计）
	SQLIncludeTruncated bool

	// CompareQuery 第二个统计项（如另一类计数日志）；非空时同时统计并算差额
	CompareQuery       string
	ComparePattern     *regexp.Regexp
	ComparePatternText string

	User     string
	Password string
	APIKey   string
	Insecure bool

	WeekStartSunday bool // Kibana 的 now/w 默认从周日开始

	DedupMode string // message（默认，按日志正文去重）| timestamp | id | none
	ShowDup   bool   // 输出被跳过的重复样本

	Chunk time.Duration // 分片窗口（默认 24h）；0 表示不分片

	PageSize    int
	MaxDocs     int
	Timeout     time.Duration
	SummaryOnly bool
	GroupBy     string // "" | hour | day
	SortDesc    bool
	ShowRaw     bool
}

type logStatsRow struct {
	Timestamp string
	Number    int64
	Raw       string
	Index     string
}

// logStatsHit 一条命中的日志文档
type logStatsHit struct {
	ID    string
	Index string
	TS    string // 时间字段（@timestamp）原值
	Raw   string // 正文字段（message）内容
	Trace string // 从正文里解析出的 traceId（MyBatis 日志配对用）
}

type logStatsGroup struct {
	Label string
	Count int
	Sum   int64
	Min   int64
	Max   int64
}

type logStatsReport struct {
	Spec        logStatsSpec
	ResolvedIdx string
	Endpoint    string

	Docs       int
	Matched    int
	DupSkipped int
	DupSamples []logStatsRow
	Partial    string
	Sum        int64
	Min        int64
	Max        int64
	HasNumber  bool
	FirstTS    string
	LastTS     string
	Rows       []logStatsRow
	Groups     map[string]*logStatsGroup
	Truncated  bool

	// SQL 批量插入汇总（--sql-table 时才有）
	SQL *logStatsAggResult

	// Compare 第二个统计项的结果（--compare 时才有）
	Compare *logStatsReport
}

// logStatsDiff 两个统计项的差额
type logStatsDiff struct {
	MainLabel    string
	CompareLabel string
	MainSum      int64
	CompareSum   int64
	MainCount    int
	CompareCount int
	HasMain      bool
	HasCompare   bool
}

// Diff 主项 - 对比项
func (d logStatsDiff) Diff() int64 { return d.MainSum - d.CompareSum }

// Ratio 对比项占主项的百分比；主项为 0 时返回 0
func (d logStatsDiff) Ratio() float64 {
	if d.MainSum == 0 {
		return 0
	}
	return float64(d.CompareSum) / float64(d.MainSum) * 100
}

func doLogStats(_ *define.RemoteMachine, c []string, outputChan chan<- string) error {
	spec, err := parseLogStatsArgs(c)
	if err != nil {
		return err
	}

	utils.SendOutput(outputChan, fmt.Sprintf("查询 %s …", spec.BaseURL))

	ctx, cancel := context.WithTimeout(context.Background(), spec.Timeout)
	defer cancel()

	report, err := runLogStats(ctx, spec)
	if err != nil {
		return err
	}
	for _, line := range formatLogStatsReport(report) {
		utils.SendOutput(outputChan, line)
	}
	return nil
}

// runLogStats 执行一次完整统计（命令与测试共用）
func runLogStats(ctx context.Context, spec logStatsSpec) (*logStatsReport, error) {
	client, err := newLogStatsClient(spec)
	if err != nil {
		return nil, err
	}

	resolved, err := client.resolveIndex(ctx)
	if err != nil {
		return nil, err
	}
	if err := client.ensureEndpoint(ctx, resolved); err != nil {
		return nil, err
	}

	report, err := collectLogStats(ctx, client, resolved, spec.Query, spec.Pattern)
	if err != nil {
		return nil, err
	}
	report.Spec = spec
	report.ResolvedIdx = resolved
	// 扫描过程中可能因 Kibana 代理超时切到 ES 直连，端点描述要在这之后再取
	report.Endpoint = client.describeEndpoint()

	// --sql-table：单独扫一次该表的 INSERT/Parameters 日志（主查询关键词里通常不含 SQL）
	if spec.SQLTable != "" {
		// Preparing 与 Parameters 是两条独立日志（Parameters 可能长达 512KB 被单独落库），
		// 所以分两次查询：先拿 Preparing 行得到列名与 traceId，再按 traceId 取 Parameters。
		var sqlHits []logStatsHit
		seen := map[string]bool{}
		addHit := func(hit logStatsHit) {
			key := hit.Index + "\x00" + hit.ID
			if hit.ID == "" {
				key = hit.Trace + "\x00" + hit.TS + "\x00" + hit.Raw[:min(64, len(hit.Raw))]
			}
			if seen[key] {
				return
			}
			seen[key] = true
			sqlHits = append(sqlHits, hit)
		}

		prepQuery := "Preparing: INSERT INTO " + spec.SQLTable
		if serr := client.scanAllFor(ctx, resolved, prepQuery, func(hit logStatsHit) bool {
			addHit(hit)
			return true
		}); serr != nil {
			return nil, fmt.Errorf("解析 %s 的 Preparing 日志失败: %w", spec.SQLTable, serr)
		}

		// 按 traceId 补取 Parameters 行
		traceIDs := collectSQLTraceIDs(sqlHits)
		for _, tid := range traceIDs {
			if serr := client.scanAllFor(ctx, resolved, tid, func(hit logStatsHit) bool {
				if strings.Contains(hit.Raw, "Parameters:") {
					addHit(hit)
				}
				return true
			}); serr != nil {
				// 单个 traceId 失败不整体失败，继续下一个
				continue
			}
		}
		if batches, _ := buildSQLBatches(sqlHits, spec.SQLTable); len(batches) > 0 {
			cols := batches[0].Columns
			amountCol := spec.SQLAmountCol
			curCol := spec.SQLCurrencyCol
			tradeCol := spec.SQLTradeCol
			if amountCol == "" {
				amountCol = pickColumn(cols, sqlAmountColumnCandidates, "amount")
			}
			if curCol == "" {
				curCol = pickColumn(cols, sqlCurrencyColumnCandidates, "currency")
			}
			if tradeCol == "" {
				tradeCol = pickColumn(cols, sqlTradeColumnCandidates, "")
			}
			report.SQL = aggregateSQLBatchesOpt(batches, amountCol, curCol, tradeCol, !spec.SQLIncludeTruncated)
			report.SQL.AmountCol = amountCol
			report.SQL.CurrencyCol = curCol
			report.SQL.TradeCol = tradeCol
		}
		report.Endpoint = client.describeEndpoint()
	}

	// 对比项（如另一类计数日志）：复用同一个客户端与时间窗，再扫一遍
	if strings.TrimSpace(spec.CompareQuery) != "" && spec.ComparePattern != nil {
		compareSpec := spec
		compareSpec.Query = spec.CompareQuery
		compareSpec.Pattern = spec.ComparePattern
		compareSpec.PatternText = spec.ComparePatternText
		compareSpec.ShowDup = false // 对比项不重复列样本

		cmp, cerr := collectLogStats(ctx, client, resolved, spec.CompareQuery, spec.ComparePattern)
		if cerr != nil {
			return nil, fmt.Errorf("统计对比项 %q 失败: %w", spec.CompareQuery, cerr)
		}
		cmp.Spec = compareSpec
		cmp.ResolvedIdx = resolved
		cmp.Endpoint = client.describeEndpoint()
		report.Compare = cmp
	}

	report.Truncated = client.truncated
	report.Partial = client.partialReason
	return report, nil
}

// collectLogStats 用一个查询/正则扫描一遍并汇总（主项与对比项共用）
func collectLogStats(ctx context.Context, client *logStatsClient, resolved, query string, pattern *regexp.Regexp) (*logStatsReport, error) {
	spec := client.spec
	report := &logStatsReport{}

	// 去重：同一条日志可能被重复采集（不同 _id、甚至不同索引），重复的不能累加
	seen := make(map[string]struct{})

	err := client.scanAllFor(ctx, resolved, query, func(hit logStatsHit) bool {
		// 展示/分组优先用日志正文里的时间戳（与 Kibana 看到的一致），
		// 因为部分服务的 @timestamp 与正文时间存在时区偏移。
		ts := hit.TS
		if spec.TimeSource != "field" {
			if local := leadingLogTimestamp(hit.Raw); local != "" {
				ts = local
			}
		}

		n, ok := extractLogStatNumber(pattern, hit.Raw)

		if spec.DedupMode != "none" {
			key := logStatsDedupKey(spec.DedupMode, hit, ts, n, ok)
			if key != "" {
				if _, dup := seen[key]; dup {
					report.DupSkipped++
					if spec.ShowDup && len(report.DupSamples) < logStatsDupSampleLimit {
						report.DupSamples = append(report.DupSamples, logStatsRow{
							Timestamp: ts,
							Number:    n,
							Raw:       hit.Raw,
							Index:     hit.Index,
						})
					}
					return true
				}
				seen[key] = struct{}{}
			}
		}

		report.Docs++
		if report.FirstTS == "" {
			report.FirstTS = ts
		}
		report.LastTS = ts

		if !ok {
			return true
		}
		report.Matched++
		report.Sum += n
		if !report.HasNumber {
			report.Min, report.Max, report.HasNumber = n, n, true
		} else {
			if n < report.Min {
				report.Min = n
			}
			if n > report.Max {
				report.Max = n
			}
		}
		report.Rows = append(report.Rows, logStatsRow{Timestamp: ts, Number: n, Raw: hit.Raw, Index: hit.Index})
		if spec.GroupBy != "" {
			addLogStatsGroup(report, spec.GroupBy, ts, n)
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return report, nil
}

// buildDiff 计算主项与对比项的差额
func buildDiff(r *logStatsReport) *logStatsDiff {
	if r == nil || r.Compare == nil {
		return nil
	}
	return &logStatsDiff{
		MainLabel:    r.Spec.Query,
		CompareLabel: r.Spec.CompareQuery,
		MainSum:      r.Sum,
		CompareSum:   r.Compare.Sum,
		MainCount:    r.Matched,
		CompareCount: r.Compare.Matched,
		HasMain:      r.HasNumber,
		HasCompare:   r.Compare.HasNumber,
	}
}

// logStatsDedupKey 生成去重键：
//   - message（默认）：日志正文归一化后比较——重复采集的同一条日志正文完全相同
//   - timestamp：正文时间戳 + 提取值，适合正文带线程名等易变字段的场景
//   - id：索引 + _id，仅去掉同一个文档重复命中的情况
func logStatsDedupKey(mode string, hit logStatsHit, ts string, n int64, hasNumber bool) string {
	switch mode {
	case "id":
		if hit.ID == "" {
			return ""
		}
		return hit.Index + "\x00" + hit.ID
	case "timestamp", "ts":
		if ts == "" && !hasNumber {
			return ""
		}
		key := ts
		if hasNumber {
			key += "\x00" + strconv.FormatInt(n, 10)
		}
		return key
	default: // message
		raw := collapseLogSpaces(hit.Raw)
		if raw == "" {
			// 没有正文时退化成文档 id，避免把空正文全当重复
			if hit.ID == "" {
				return ""
			}
			return hit.Index + "\x00" + hit.ID
		}
		return raw
	}
}

// collapseLogSpaces 去掉首尾空白并把连续空白压成一个空格，容忍采集时的排版差异
func collapseLogSpaces(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func addLogStatsGroup(report *logStatsReport, unit, ts string, n int64) {
	key := groupKeyFor(unit, ts)
	if key == "" {
		return
	}
	if report.Groups == nil {
		report.Groups = make(map[string]*logStatsGroup)
	}
	g := report.Groups[key]
	if g == nil {
		g = &logStatsGroup{Label: key, Min: n, Max: n}
		report.Groups[key] = g
	}
	g.Count++
	g.Sum += n
	if n < g.Min {
		g.Min = n
	}
	if n > g.Max {
		g.Max = n
	}
}

// groupKeyFor 把时间戳归到小时/天
func groupKeyFor(unit, ts string) string {
	t, ok := parseLogTimestamp(ts)
	if !ok {
		return ""
	}
	t = t.Local()
	if strings.EqualFold(strings.TrimSpace(unit), "day") {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:00")
}

func extractLogStatNumber(pattern *regexp.Regexp, raw string) (int64, bool) {
	if pattern == nil || raw == "" {
		return 0, false
	}
	m := pattern.FindStringSubmatch(raw)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(m[1]), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// 日志行首时间戳，例如 "2026-11-25 02:00:00.159 ..."
var leadingTimestampPattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d{1,3})?)`)

// leadingLogTimestamp 取日志正文自带的时间戳（Kibana 里看到的那个），没有则返回空
func leadingLogTimestamp(raw string) string {
	m := leadingTimestampPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(m) < 2 {
		return ""
	}
	return strings.ReplaceAll(m[1], ",", ".")
}

var logTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.000Z0700",
	"2006-01-02T15:04:05.000",
	"2006-01-02 15:04:05.000",
	"2006-01-02 15:04:05,000",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseLogTimestamp(raw string) (time.Time, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range logTimeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// formatLogStatsTimestamp 统一按本机时区显示，与 Kibana 上看到的时间一致
func formatLogStatsTimestamp(raw string) string {
	if t, ok := parseLogTimestamp(raw); ok {
		return t.Local().Format("2006-01-02 15:04:05.000")
	}
	return raw
}

// ---------------------------------------------------------------- 参数解析

func parseLogStatsArgs(c []string) (logStatsSpec, error) {
	spec := logStatsSpec{
		Field:      logStatsDefaultField,
		TimeField:  logStatsDefaultTimeField,
		TimeSource: "message",
		QueryMode:  "phrase",
		// Kibana 的 Discover 日历按周日开始一周（Nov 22 周日 ~ Nov 28 周六）
		WeekStartSunday: true,
		DedupMode:       logStatsDedupDefault,
		Chunk:           logStatsDefaultChunk,
		PageSize:        logStatsDefaultPageSize,
		MaxDocs:         logStatsDefaultMaxDocs,
		Timeout:         logStatsDefaultTimeout,
		ToExpr:          "now",
		FromExpr:        "now-7d",
	}

	lastArg := ""
	fromFlag := ""
	toFlag := ""
	for i := 1; i < len(c); i++ {
		tok := strings.TrimSpace(c[i])
		if tok == "" {
			continue
		}
		if !strings.HasPrefix(tok, "--") {
			if spec.RequestURL == "" {
				spec.RequestURL = trimArgQuotes(tok)
				continue
			}
			return spec, fmt.Errorf("参数错误: 多余的位置参数 %q", tok)
		}

		name := strings.TrimPrefix(tok, "--")
		value := ""
		hasValue := false
		if idx := strings.Index(name, "="); idx >= 0 {
			value = name[idx+1:]
			name = name[:idx]
			hasValue = true
		}
		if !hasValue && i+1 < len(c) && !strings.HasPrefix(strings.TrimSpace(c[i+1]), "--") {
			value = trimArgQuotes(c[i+1])
			i++
		} else {
			value = trimArgQuotes(value)
		}

		switch strings.ToLower(name) {
		case "url", "kibana", "kibana-url":
			spec.RequestURL = value
		case "es-url", "es":
			spec.ESURL = value
		case "index":
			spec.Index = value
		case "from":
			fromFlag = value
		case "to":
			toFlag = value
		case "last", "since":
			lastArg = value
		case "range", "period", "区间":
			spec.RangePreset = value
		case "query", "q", "search", "keyword":
			spec.Query = value
		case "query-mode", "mode":
			spec.QueryMode = strings.ToLower(strings.TrimSpace(value))
		case "field":
			spec.Field = value
		case "time-field":
			spec.TimeField = value
		case "dedup", "dedupe":
			v := strings.ToLower(strings.TrimSpace(value))
			switch v {
			case "", "on", "message", "msg", "auto":
				spec.DedupMode = logStatsDedupDefault
			case "timestamp", "ts":
				spec.DedupMode = "timestamp"
			case "id":
				spec.DedupMode = "id"
			case "off", "none", "no":
				spec.DedupMode = "none"
			default:
				return spec, errors.New("参数错误: --dedup 只支持 message / timestamp / id / none")
			}
		case "chunk", "window":
			v := strings.ToLower(strings.TrimSpace(value))
			if v == "" || v == "0" || v == "none" || v == "off" {
				spec.Chunk = 0
				break
			}
			d, err := parseKibanaDuration(v)
			if err != nil || d <= 0 {
				return spec, fmt.Errorf("参数错误: --chunk 无效: %s", value)
			}
			spec.Chunk = d
		case "show-dup", "show-dup-samples":
			spec.ShowDup = true
		case "week-start":
			v := strings.ToLower(strings.TrimSpace(value))
			switch v {
			case "sunday", "sun", "0":
				spec.WeekStartSunday = true
			case "monday", "mon", "1", "iso":
				spec.WeekStartSunday = false
			default:
				return spec, errors.New("参数错误: --week-start 只支持 sunday / monday")
			}
		case "time-source":
			v := strings.ToLower(strings.TrimSpace(value))
			if v != "message" && v != "field" {
				return spec, errors.New("参数错误: --time-source 只支持 message / field")
			}
			spec.TimeSource = v
		case "pattern", "regex":
			spec.PatternText = value
		case "sql-table", "sql":
			spec.SQLTable = normalizeSQLTable(value)
		case "sql-amount-column":
			spec.SQLAmountCol = strings.ToLower(strings.TrimSpace(value))
		case "sql-currency-column":
			spec.SQLCurrencyCol = strings.ToLower(strings.TrimSpace(value))
		case "sql-include-truncated":
			spec.SQLIncludeTruncated = true
		case "sql-trade-column":
			spec.SQLTradeCol = strings.ToLower(strings.TrimSpace(value))
		case "compare", "compare-query", "compare-with":
			spec.CompareQuery = value
		case "compare-pattern":
			spec.ComparePatternText = value
		case "user":
			if strings.Contains(value, ":") {
				parts := strings.SplitN(value, ":", 2)
				spec.User, spec.Password = parts[0], parts[1]
			} else {
				spec.User = value
			}
		case "password", "pass", "pwd":
			spec.Password = value
		case "api-key", "apikey", "token":
			spec.APIKey = value
		case "insecure", "skip-verify":
			spec.Insecure = true
		case "page-size", "size":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return spec, fmt.Errorf("参数错误: --page-size 无效: %s", value)
			}
			spec.PageSize = n
		case "max-docs", "limit":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return spec, fmt.Errorf("参数错误: --max-docs 无效: %s", value)
			}
			spec.MaxDocs = n
		case "timeout":
			d, err := parseKibanaDuration(value)
			if err != nil || d <= 0 {
				return spec, fmt.Errorf("参数错误: --timeout 无效: %s", value)
			}
			spec.Timeout = d
		case "summary-only", "quiet":
			spec.SummaryOnly = true
		case "group", "group-by":
			g := strings.ToLower(strings.TrimSpace(value))
			if g != "" && g != "hour" && g != "day" {
				return spec, errors.New("参数错误: --group 只支持 hour / day")
			}
			spec.GroupBy = g
		case "desc":
			spec.SortDesc = true
		case "show-raw", "raw":
			spec.ShowRaw = true
		case "help", "h":
			return spec, errors.New(logStatsUsage())
		default:
			return spec, fmt.Errorf("参数错误: 未知选项 --%s\n%s", name, logStatsUsage())
		}
	}

	if strings.TrimSpace(spec.RequestURL) == "" {
		return spec, errors.New(logStatsUsage())
	}

	// 从 Kibana URL 抽取 basePath / 索引 / 查询 / 时间范围（显式参数优先）
	state, err := parseKibanaDiscoverURL(spec.RequestURL)
	if err != nil {
		return spec, err
	}
	spec.BaseURL = state.BaseURL
	if spec.Index == "" {
		spec.Index = state.Index
	}
	if spec.Query == "" {
		spec.Query = state.Query
	}
	// 时间范围优先级：--from/--to > --last > --range > URL > 默认
	fromExpr, toExpr := spec.FromExpr, spec.ToExpr
	if state.From != "" {
		fromExpr = state.From
	}
	if state.To != "" {
		toExpr = state.To
	}
	if spec.RangePreset != "" {
		preset, ok := resolveLogStatsRange(spec.RangePreset)
		if !ok {
			return spec, fmt.Errorf("参数错误: 未知区间 %q\n可用区间: %s", spec.RangePreset, logStatsRangeNames())
		}
		fromExpr, toExpr = preset.From, preset.To
		spec.RangeLabel = preset.Label
	}
	if fromFlag != "" {
		fromExpr = fromFlag
	}
	if toFlag != "" {
		toExpr = toFlag
	}
	spec.FromExpr, spec.ToExpr = fromExpr, toExpr
	if spec.QueryMode == "phrase" && strings.EqualFold(state.Language, "lucene") {
		spec.QueryMode = "query_string"
	}

	if strings.TrimSpace(spec.Query) == "" {
		return spec, errors.New("参数错误: 缺少搜索内容，用 --query 指定，或使用带 query 的 Kibana 地址")
	}
	if strings.TrimSpace(spec.Index) == "" {
		return spec, errors.New("参数错误: 缺少索引，用 --index 指定，或使用带 index 的 Kibana 地址")
	}

	now := time.Now()
	if lastArg != "" {
		d, err := parseKibanaDuration(lastArg)
		if err != nil {
			return spec, fmt.Errorf("参数错误: --last 无效: %s", lastArg)
		}
		spec.From, spec.To = now.Add(-d).Truncate(time.Second), now.Truncate(time.Second)
	} else {
		from, err := parseKibanaTimeExpr(spec.FromExpr, now, kibanaTimeOpts{
			roundUp:         false,
			weekStartSunday: spec.WeekStartSunday,
		})
		if err != nil {
			return spec, fmt.Errorf("参数错误: --from 无效: %v", err)
		}
		to, err := parseKibanaTimeExpr(spec.ToExpr, now, kibanaTimeOpts{
			roundUp:         true,
			weekStartSunday: spec.WeekStartSunday,
		})
		if err != nil {
			return spec, fmt.Errorf("参数错误: --to 无效: %v", err)
		}
		spec.From, spec.To = from, to
	}
	if !spec.To.After(spec.From) {
		return spec, fmt.Errorf("参数错误: 时间范围为空（from=%s to=%s）",
			spec.From.Format(time.DateTime), spec.To.Format(time.DateTime))
	}

	if spec.PatternText == "" {
		spec.PatternText = logStatsDefaultPattern
	}
	re, err := regexp.Compile(spec.PatternText)
	if err != nil {
		return spec, fmt.Errorf("参数错误: --pattern 正则无效: %v", err)
	}
	spec.Pattern = re

	if spec.Field == "" {
		spec.Field = logStatsDefaultField
	}
	if spec.TimeField == "" {
		spec.TimeField = logStatsDefaultTimeField
	}

	// 对比项：给了 --compare 才启用
	if strings.TrimSpace(spec.CompareQuery) != "" {
		if spec.ComparePatternText == "" {
			spec.ComparePatternText = logStatsDefaultComparePattern
		}
		cre, err := regexp.Compile(spec.ComparePatternText)
		if err != nil {
			return spec, fmt.Errorf("参数错误: --compare-pattern 正则无效: %v", err)
		}
		spec.ComparePattern = cre
	}
	return spec, nil
}

// trimArgQuotes 去掉任务步骤里手写的包裹引号（步骤文本只按空白切分，不经过 shell）
func trimArgQuotes(s string) string {
	t := strings.TrimSpace(s)
	if len(t) >= 2 {
		if (t[0] == '"' && t[len(t)-1] == '"') || (t[0] == '\'' && t[len(t)-1] == '\'') {
			return t[1 : len(t)-1]
		}
	}
	return t
}

func logStatsUsage() string {
	return "用法: log-stats <kibana或es地址> [--query 搜索内容] [--range 区间] [--from 时间] [--to 时间] [--last 7d]\n" +
		"  区间: 本周/上周/本月/上月/今年/今天/昨天/近7天/近14天/近30天/近2月/近3月/近半年/近1年\n" +
		"        英文同义 this-week/last-week/this-month/last-month/this-year/today/yesterday/last-7d/last-2m，也支持 last-Nd/last-Nw/last-Nm/last-Ny\n" +
		"  [--index 索引] [--field message] [--time-field @timestamp] [--time-source message|field]\n" +
		"  [--pattern 正则] [--compare 对比搜索内容] [--compare-pattern 对比正则]\n" +
		"  [--query-mode phrase|match|term|query_string] [--user 用户:密码] [--api-key KEY]\n" +
		"  [--es-url http://host:9200] [--insecure] [--page-size 1000] [--max-docs 200000]\n" +
		"  [--timeout 120s] [--group hour|day] [--desc] [--show-raw] [--summary-only]\n" +
		"  [--week-start sunday|monday]（默认 sunday，与 Kibana 日历一致）\n" +
		"  [--dedup message|timestamp|id|none]（默认 message：重复采集的同一条日志只算一次）\n" +
		"  [--show-dup] 列出被跳过的重复样本\n" +
		"  [--sql-table 表名] 解析该表 MyBatis 批量 INSERT，按金额/币种汇总\n" +
		"  [--sql-amount-column amount] [--sql-currency-column currency] [--sql-trade-column trade_type]\n" +
		"  [--sql-include-truncated] 截断批次也计入（默认跳过，避免用部分数据冒充合计）\n" +
		"  [--chunk 24h] 分片窗口（默认按天；范围很大时可调小，或 0 关闭分片）\n" +
		"例: log-stats \"http://kibana.example.com:5601/app/discover#/?_g=(time:(from:now%2Fw,to:now%2Fw))" +
		"&_a=(index:'00000000-0000-0000-0000-000000000000',query:(language:kuery,query:%22some+metric%22))\""
}

// ---------------------------------------------------------------- Kibana URL

type kibanaDiscoverState struct {
	BaseURL  string
	Index    string
	Query    string
	Language string
	From     string
	To       string
}

var dataViewIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// parseKibanaDiscoverURL 解析 Discover 地址里的 basePath / index / query / 时间范围
func parseKibanaDiscoverURL(raw string) (kibanaDiscoverState, error) {
	var st kibanaDiscoverState
	s := strings.TrimSpace(raw)
	if s == "" {
		return st, errors.New("参数错误: 请求地址为空")
	}
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return st, fmt.Errorf("参数错误: 请求地址无效: %s", raw)
	}

	basePath := ""
	if idx := strings.Index(u.Path, "/app/"); idx >= 0 {
		basePath = strings.TrimSuffix(u.Path[:idx], "/")
	} else {
		basePath = strings.TrimSuffix(u.Path, "/")
	}
	st.BaseURL = fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, basePath)

	fragment := u.Fragment
	if fragment == "" {
		fragment = u.RawQuery
	}
	if fragment == "" {
		return st, nil
	}

	if block, ok := risonBlock(fragment, "time"); ok {
		st.From = risonValue(block, "from")
		st.To = risonValue(block, "to")
	}
	st.Index = risonValue(fragment, "index")
	if block, ok := risonBlock(fragment, "query"); ok {
		st.Language = risonValue(block, "language")
		st.Query = risonValue(block, "query")
	} else {
		st.Query = risonValue(fragment, "query")
	}
	return st, nil
}

// risonBlock 取形如 key:(...) 的括号内容
func risonBlock(s, key string) (string, bool) {
	re := regexp.MustCompile(`(?:^|[,(!\[])\s*` + regexp.QuoteMeta(key) + `\s*:\s*\(`)
	loc := re.FindStringIndex(s)
	if loc == nil {
		return "", false
	}
	start := loc[1]
	depth := 1
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[start:i], true
			}
		}
	}
	return "", false
}

// risonValue 取形如 key:'value' / key:value / key:"value" 的标量值
func risonValue(s, key string) string {
	re := regexp.MustCompile(`(?:^|[,(!\[])\s*` + regexp.QuoteMeta(key) + `\s*:\s*`)
	loc := re.FindStringIndex(s)
	if loc == nil {
		return ""
	}
	rest := strings.TrimLeft(s[loc[1]:], " ")
	if rest == "" {
		return ""
	}
	if rest[0] == '\'' || rest[0] == '"' {
		quote := rest[0]
		var b strings.Builder
		for i := 1; i < len(rest); i++ {
			if rest[i] == '!' && i+1 < len(rest) {
				i++ // Rison 用 ! 转义下一个字符
				b.WriteByte(rest[i])
				continue
			}
			if rest[i] == quote {
				break
			}
			b.WriteByte(rest[i])
		}
		return strings.TrimSpace(b.String())
	}
	if rest[0] == '(' {
		return ""
	}
	end := strings.IndexAny(rest, ",) ")
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

// ---------------------------------------------------------------- 区间预设

// logStatsRange 命名区间：用 Kibana 时间表达式表达，复用同一套取整逻辑
type logStatsRange struct {
	From  string
	To    string
	Label string
}

// 常用区间预设：中文写法与英文写法等价，键统一小写
var logStatsRangePresets = map[string]logStatsRange{
	"today":      {"now/d", "now/d", "今天"},
	"今天":         {"now/d", "now/d", "今天"},
	"yesterday":  {"now-1d/d", "now-1d/d", "昨天"},
	"昨天":         {"now-1d/d", "now-1d/d", "昨天"},
	"this-week":  {"now/w", "now/w", "本周"},
	"本周":         {"now/w", "now/w", "本周"},
	"这周":         {"now/w", "now/w", "本周"},
	"这个周":        {"now/w", "now/w", "本周"},
	"这个星期":       {"now/w", "now/w", "本周"},
	"last-week":  {"now-1w/w", "now-1w/w", "上周"},
	"上周":         {"now-1w/w", "now-1w/w", "上周"},
	"上个周":        {"now-1w/w", "now-1w/w", "上周"},
	"上个星期":       {"now-1w/w", "now-1w/w", "上周"},
	"this-month": {"now/M", "now/M", "本月"},
	"本月":         {"now/M", "now/M", "本月"},
	"这个月":        {"now/M", "now/M", "本月"},
	"last-month": {"now-1M/M", "now-1M/M", "上月"},
	"上月":         {"now-1M/M", "now-1M/M", "上月"},
	"上个月":        {"now-1M/M", "now-1M/M", "上月"},
	"this-year":  {"now/y", "now/y", "今年"},
	"今年":         {"now/y", "now/y", "今年"},
	"last-year":  {"now-1y/y", "now-1y/y", "去年"},
	"去年":         {"now-1y/y", "now-1y/y", "去年"},
	"last-7d":    {"now-7d", "now", "近 7 天"},
	"近7天":        {"now-7d", "now", "近 7 天"},
	"最近7天":       {"now-7d", "now", "近 7 天"},
	"last-14d":   {"now-14d", "now", "近 14 天"},
	"近14天":       {"now-14d", "now", "近 14 天"},
	"last-30d":   {"now-30d", "now", "近 30 天"},
	"近30天":       {"now-30d", "now", "近 30 天"},
	"近1月":        {"now-1M", "now", "近 1 月"},
	"last-1m":    {"now-1M", "now", "近 1 月"},
	"last-2m":    {"now-2M", "now", "近 2 月"},
	"近2月":        {"now-2M", "now", "近 2 月"},
	"最近2月":       {"now-2M", "now", "近 2 月"},
	"last-3m":    {"now-3M", "now", "近 3 月"},
	"近3月":        {"now-3M", "now", "近 3 月"},
	"last-6m":    {"now-6M", "now", "近 6 月"},
	"近6月":        {"now-6M", "now", "近 6 月"},
	"近半年":        {"now-6M", "now", "近 6 月"},
	"last-1y":    {"now-1y", "now", "近 1 年"},
	"近1年":        {"now-1y", "now", "近 1 年"},
}

var (
	rangeGenericEN = regexp.MustCompile(`^last-(\d+)\s*(h|hour|hours|d|day|days|w|week|weeks|m|mo|month|months|y|year|years)$`)
	rangeGenericCN = regexp.MustCompile(`^近(\d+|[两二])\s*个?\s*(小时|天|日|周|月|年)$`)
)

var rangeUnitCN = map[byte]string{
	'h': "小时",
	'd': "天",
	'w': "周",
	'M': "月",
	'y': "年",
}

// resolveLogStatsRange 把 --range 的值解析成 Kibana 时间表达式
func resolveLogStatsRange(raw string) (logStatsRange, bool) {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.ReplaceAll(key, " ", "")
	if key == "" {
		return logStatsRange{}, false
	}
	if preset, ok := logStatsRangePresets[key]; ok {
		return preset, true
	}

	// 通用写法：last-2m / last-3w / last-12h
	if m := rangeGenericEN.FindStringSubmatch(key); len(m) == 3 {
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 {
			return logStatsRange{}, false
		}
		unit, ok := normalizeRangeUnit(m[2])
		if !ok {
			return logStatsRange{}, false
		}
		return logStatsRange{
			From:  fmt.Sprintf("now-%d%c", n, unit),
			To:    "now",
			Label: fmt.Sprintf("近 %d %s", n, rangeUnitCN[unit]),
		}, true
	}

	// 中文通用写法：近2月 / 近3个月 / 近两天 / 近1年
	if m := rangeGenericCN.FindStringSubmatch(key); len(m) == 3 {
		n, err := strconv.Atoi(strings.NewReplacer("两", "2", "二", "2").Replace(m[1]))
		if err != nil || n < 1 {
			return logStatsRange{}, false
		}
		unit, ok := normalizeRangeUnitCN(m[2])
		if !ok {
			return logStatsRange{}, false
		}
		return logStatsRange{
			From:  fmt.Sprintf("now-%d%c", n, unit),
			To:    "now",
			Label: fmt.Sprintf("近 %d %s", n, rangeUnitCN[unit]),
		}, true
	}

	if preset, ok := logStatsRangePresets[key]; ok {
		return preset, true
	}
	if r, ok := resolveRollingRange(key); ok {
		return r, true
	}
	return logStatsRange{}, false
}

// resolveRollingRange 支持 last-<n><unit> 之外的惯用写法（用于扩展）
func resolveRollingRange(key string) (logStatsRange, bool) {
	return logStatsRange{}, false
}

func normalizeRangeUnit(unit string) (byte, bool) {
	switch strings.ToLower(unit) {
	case "h", "hour", "hours":
		return 'h', true
	case "d", "day", "days":
		return 'd', true
	case "w", "week", "weeks":
		return 'w', true
	case "m", "mo", "month", "months":
		return 'M', true // 区间语境下 m 表示“月”，分钟请用 --last 2m
	case "y", "year", "years":
		return 'y', true
	default:
		return 0, false
	}
}

func normalizeRangeUnitCN(unit string) (byte, bool) {
	switch unit {
	case "小时":
		return 'h', true
	case "天", "日":
		return 'd', true
	case "周":
		return 'w', true
	case "月":
		return 'M', true
	case "年":
		return 'y', true
	default:
		return 0, false
	}
}

// logStatsRangeNames 出错时提示可用区间
func logStatsRangeNames() string {
	names := make([]string, 0, len(logStatsRangePresets))
	for k := range logStatsRangePresets {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ") + "；也支持近N天/近N月/近N年 与 last-Nd/last-Nm/last-Ny"
}

// ---------------------------------------------------------------- 时间解析

var kibanaTimeUnit = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': 24 * time.Hour,
	'w': 7 * 24 * time.Hour,
}

// parseKibanaDuration 解析 7d / 24h / 30m / 1w / 60s
func parseKibanaDuration(expr string) (time.Duration, error) {
	e := strings.TrimSpace(expr)
	for strings.HasPrefix(e, "-") || strings.HasPrefix(e, "+") {
		e = strings.TrimSpace(e[1:])
	}
	if e == "" {
		return 0, errors.New("空时间跨度")
	}
	numEnd := 0
	for numEnd < len(e) && e[numEnd] >= '0' && e[numEnd] <= '9' {
		numEnd++
	}
	if numEnd == 0 {
		return 0, fmt.Errorf("缺少数字: %s", expr)
	}
	n, err := strconv.Atoi(e[:numEnd])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("数字无效: %s", expr)
	}
	unit := strings.ToLower(strings.TrimSpace(e[numEnd:]))
	switch unit {
	case "s", "sec", "second", "seconds":
		return time.Duration(n) * time.Second, nil
	case "m", "min", "minute", "minutes":
		return time.Duration(n) * time.Minute, nil
	case "h", "hour", "hours":
		return time.Duration(n) * time.Hour, nil
	case "d", "day", "days":
		return time.Duration(n) * 24 * time.Hour, nil
	case "w", "week", "weeks":
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case "mo", "month", "months":
		return time.Duration(n) * 30 * 24 * time.Hour, nil
	case "y", "year", "years":
		return time.Duration(n) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("未知单位: %s", expr)
	}
}

type kibanaTimeOpts struct {
	roundUp         bool
	weekStartSunday bool
}

// parseKibanaTimeExpr 支持 now、now-7d、now/w、now-1w/d、now-2M 与绝对时间
func parseKibanaTimeExpr(expr string, now time.Time, opts kibanaTimeOpts) (time.Time, error) {
	e := strings.TrimSpace(expr)
	if e == "" {
		return time.Time{}, errors.New("空时间表达式")
	}
	if !strings.HasPrefix(strings.ToLower(e), "now") {
		if t, ok := parseLogTimestamp(e); ok {
			return t, nil
		}
		return time.Time{}, fmt.Errorf("无法识别的时间: %s", expr)
	}

	// 统一截到秒，避免出现 21:10:40.126 这种毛刺（Kibana 显示 .000）
	t := now.Truncate(time.Second)
	rest := e[3:]
	if strings.TrimSpace(rest) == "" {
		return t, nil
	}
	for i := 0; i < len(rest); {
		switch rest[i] {
		case '/':
			if i+1 >= len(rest) {
				return time.Time{}, fmt.Errorf("时间表达式缺少单位: %s", expr)
			}
			unit := rest[i+1]
			if unit == 'M' {
				t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
				if opts.roundUp {
					t = t.AddDate(0, 1, 0).Add(-time.Millisecond)
				}
			} else {
				d, err := kibanaTimeUnitDuration(unit)
				if err != nil {
					return time.Time{}, err
				}
				floored := roundDownTo(t, d, opts.weekStartSunday)
				if opts.roundUp {
					// 与 Kibana 一致：取周期最后一毫秒（如 Nov 28 23:59:59.999）
					t = floored.Add(d).Add(-time.Millisecond)
				} else {
					t = floored
				}
			}
			i += 2
		case '+', '-':
			sign := 1
			if rest[i] == '-' {
				sign = -1
			}
			j := i + 1
			startDigits := j
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			if startDigits == j {
				return time.Time{}, fmt.Errorf("时间表达式缺少数字: %s", expr)
			}
			n, err := strconv.Atoi(rest[startDigits:j])
			if err != nil {
				return time.Time{}, err
			}
			if j >= len(rest) {
				return time.Time{}, fmt.Errorf("时间表达式缺少单位: %s", expr)
			}
			unit := rest[j]
			switch unit {
			case 'M':
				t = t.AddDate(0, sign*n, 0)
			case 'y':
				t = t.AddDate(sign*n, 0, 0)
			default:
				d, err := kibanaTimeUnitDuration(unit)
				if err != nil {
					return time.Time{}, err
				}
				t = t.Add(time.Duration(sign*n) * d)
			}
			i = j + 1
		default:
			return time.Time{}, fmt.Errorf("时间表达式无法识别: %s", expr)
		}
	}
	return t, nil
}

func kibanaTimeUnitDuration(unit byte) (time.Duration, error) {
	d, ok := kibanaTimeUnit[strings.ToLower(string(unit))[0]]
	if !ok {
		return 0, fmt.Errorf("未知时间单位: %c", unit)
	}
	return d, nil
}

func roundDownTo(t time.Time, unit time.Duration, weekStartSunday bool) time.Time {
	if unit <= 0 {
		return t
	}
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	if unit == 7*24*time.Hour {
		weekday := int(midnight.Weekday()) // Sunday = 0
		if weekStartSunday {
			return midnight.AddDate(0, 0, -weekday)
		}
		// ISO 周：周一为一周起点
		if weekday == 0 {
			weekday = 7
		}
		return midnight.AddDate(0, 0, -(weekday - 1))
	}
	offset := t.Sub(midnight)
	return midnight.Add(offset - offset%unit)
}

// ---------------------------------------------------------------- HTTP

type logStatsClient struct {
	spec          logStatsSpec
	http          *http.Client
	endpoint      string // kibana-proxy | es
	endpointNote  string
	truncated     bool
	partialReason string
}

// logStatsHTTPError 带上状态码与响应体，便于判断是否 Kibana 代理超时
type logStatsHTTPError struct {
	Endpoint string
	Status   int
	Body     string
	Cause    error
}

func (e *logStatsHTTPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s 请求失败: %v", e.Endpoint, e.Cause)
	}
	return fmt.Sprintf("%s 返回 HTTP %d: %s", e.Endpoint, e.Status, snippet([]byte(e.Body)))
}

// proxyTimeout Kibana console proxy 自带 30s 上限，超时会回 502 Client request timeout
func (e *logStatsHTTPError) proxyTimeout() bool {
	if e.Endpoint != "kibana-proxy" {
		return false
	}
	if e.Status == http.StatusGatewayTimeout || e.Status == http.StatusBadGateway {
		return true
	}
	msg := strings.ToLower(e.Body)
	if e.Cause != nil {
		msg += " " + strings.ToLower(e.Cause.Error())
	}
	for _, hint := range []string{
		"client request timeout",
		"context deadline exceeded",
		"timeout awaiting response headers",
		"i/o timeout",
	} {
		if strings.Contains(msg, hint) {
			return true
		}
	}
	return false
}

func newLogStatsClient(spec logStatsSpec) (*logStatsClient, error) {
	if spec.BaseURL == "" {
		return nil, errors.New("请求地址无效")
	}
	transport := &http.Transport{
		MaxIdleConns:    8,
		IdleConnTimeout: 30 * time.Second,
		// 不能短于 Kibana console proxy 自身的 30s 上限，
		// 否则拿不到它的 502 "Client request timeout"，就没法触发 ES 直连回退
		ResponseHeaderTimeout: 75 * time.Second,
	}
	if spec.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // 用户显式要求跳过证书校验
	}
	endpoint := "kibana-proxy"
	if strings.TrimSpace(spec.ESURL) != "" {
		endpoint = "es"
	}
	return &logStatsClient{
		spec:     spec,
		endpoint: endpoint,
		http:     &http.Client{Transport: transport, Timeout: spec.Timeout},
	}, nil
}

func (c *logStatsClient) describeEndpoint() string {
	desc := "Kibana 代理: " + c.spec.BaseURL
	if c.endpoint == "es" {
		desc = "Elasticsearch 直连: " + strings.TrimSuffix(c.spec.ESURL, "/")
	}
	if c.endpointNote != "" {
		desc += "（" + c.endpointNote + "）"
	}
	return desc
}

func (c *logStatsClient) authHeaders(req *http.Request) {
	if c.spec.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.spec.APIKey)
		return
	}
	if c.spec.User != "" {
		token := base64.StdEncoding.EncodeToString([]byte(c.spec.User + ":" + c.spec.Password))
		req.Header.Set("Authorization", "Basic "+token)
	}
}

// doRequest 发送请求并读取响应
func (c *logStatsClient) doRequest(ctx context.Context, method, target string, body any) ([]byte, int, error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("序列化请求失败: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("kbn-xsrf", "true")
	req.Header.Set("x-elastic-internal-origin", "Kibana")
	c.authHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("读取响应失败: %w", err)
	}
	return data, resp.StatusCode, nil
}

// kibanaCall 调 Kibana 自身接口（数据视图等），直连 Kibana 地址
func (c *logStatsClient) kibanaCall(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	target := strings.TrimSuffix(c.spec.BaseURL, "/") + path
	return c.doRequest(ctx, method, target, body)
}

// call 调 Elasticsearch。kibana 模式下走 /api/console/proxy 转发，es 模式直连。
// path 以 / 开头，例如 "/log-*/_search?scroll=2m"
func (c *logStatsClient) call(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var target string
	if c.endpoint == "es" {
		target = strings.TrimSuffix(c.spec.ESURL, "/") + path
	} else {
		rel := strings.TrimPrefix(path, "/")
		target = strings.TrimSuffix(c.spec.BaseURL, "/") +
			"/api/console/proxy?path=" + url.QueryEscape(rel) + "&method=" + method
	}
	return c.doRequest(ctx, method, target, body)
}

// resolveIndex 把 data view id 换成真实索引模式
func (c *logStatsClient) resolveIndex(ctx context.Context) (string, error) {
	idx := strings.TrimSpace(c.spec.Index)
	if idx == "" {
		return "", errors.New("索引为空")
	}
	isDataViewID := dataViewIDPattern.MatchString(idx)
	if c.endpoint == "es" {
		if isDataViewID {
			return "", fmt.Errorf("直连 ES 模式下 %s 是 Kibana data view id，请用 --index 指定真实索引模式（如 log-*）", idx)
		}
		return idx, nil
	}
	if !isDataViewID {
		return idx, nil
	}

	// Kibana 7.11+
	if data, status, err := c.kibanaCall(ctx, http.MethodGet, "/api/data_views/data_view/"+url.PathEscape(idx), nil); err == nil && status < 300 {
		var out struct {
			DataView struct {
				Title string `json:"title"`
			} `json:"data_view"`
		}
		if json.Unmarshal(data, &out) == nil && strings.TrimSpace(out.DataView.Title) != "" {
			return out.DataView.Title, nil
		}
	}

	// Kibana 7.0–7.10
	if data, status, err := c.kibanaCall(ctx, http.MethodGet, "/api/saved_objects/index-pattern/"+url.PathEscape(idx), nil); err == nil && status < 300 {
		var out struct {
			Attributes struct {
				Title string `json:"title"`
			} `json:"attributes"`
		}
		if json.Unmarshal(data, &out) == nil && strings.TrimSpace(out.Attributes.Title) != "" {
			return out.Attributes.Title, nil
		}
	}

	return "", fmt.Errorf("无法通过 Kibana 解析 data view %s 对应的索引，请用 --index 指定索引模式（如 log-*）", idx)
}

// switchToES 退化到直连 ES：优先用户给的 --es-url，否则同一主机的 9200
func (c *logStatsClient) switchToES(note string) error {
	if strings.TrimSpace(c.spec.ESURL) == "" {
		base, err := url.Parse(c.spec.BaseURL)
		if err != nil {
			return fmt.Errorf("无法推导 ES 地址: %v", err)
		}
		c.spec.ESURL = fmt.Sprintf("%s://%s:9200", base.Scheme, base.Hostname())
	}
	c.endpoint = "es"
	c.endpointNote = note
	return nil
}

// ensureEndpoint 先探一次 Kibana 代理；不可用（含代理 30s 上限）时自动改走 ES 直连
func (c *logStatsClient) ensureEndpoint(ctx context.Context, index string) error {
	probe := map[string]any{
		"size":  0,
		"query": map[string]any{"match_all": map[string]any{}},
	}
	data, status, err := c.call(ctx, http.MethodPost, "/"+index+"/_search", probe)
	if err == nil && status < 300 {
		return nil
	}
	firstErr := err
	if firstErr == nil {
		firstErr = fmt.Errorf("HTTP %d: %s", status, snippet(data))
	}
	if c.endpoint == "es" {
		return fmt.Errorf("Elasticsearch 探测失败: %v", firstErr)
	}

	if serr := c.switchToES("Kibana 代理不可用，已自动切换"); serr != nil {
		return fmt.Errorf("Kibana 代理不可用（%v），%v", firstErr, serr)
	}

	data2, status2, err2 := c.call(ctx, http.MethodPost, "/"+index+"/_search", probe)
	if err2 == nil && status2 < 300 {
		return nil
	}
	if err2 != nil {
		return fmt.Errorf("Kibana 代理不可用（%v），直连 ES %s 也失败: %v", firstErr, c.spec.ESURL, err2)
	}
	return fmt.Errorf("Kibana 代理不可用（%v），直连 ES %s 返回 HTTP %d: %s", firstErr, c.spec.ESURL, status2, snippet(data2))
}

func (c *logStatsClient) buildQuery() map[string]any {
	return c.buildQueryFor(c.spec.From, c.spec.To)
}

func (c *logStatsClient) buildQueryFor(from, to time.Time) map[string]any {
	return c.buildQueryForQuery(from, to, c.spec.Query)
}

// buildQueryForQuery 用指定搜索内容构造查询（对比项用不同关键词）
func (c *logStatsClient) buildQueryForQuery(from, to time.Time, query string) map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"filter": []any{
				map[string]any{
					"range": map[string]any{
						c.spec.TimeField: map[string]any{
							"gte": from.Format(time.RFC3339Nano),
							"lte": to.Format(time.RFC3339Nano),
						},
					},
				},
			},
			"must": []any{logStatsQueryClauseFor(c.spec, query)},
		},
	}
}

func logStatsQueryClause(spec logStatsSpec) map[string]any {
	return logStatsQueryClauseFor(spec, spec.Query)
}

func logStatsQueryClauseFor(spec logStatsSpec, q string) map[string]any {
	switch strings.ToLower(strings.TrimSpace(spec.QueryMode)) {
	case "match":
		return map[string]any{"match": map[string]any{spec.Field: q}}
	case "term":
		return map[string]any{"term": map[string]any{spec.Field: q}}
	case "query_string", "querystring", "lucene":
		return map[string]any{
			"query_string": map[string]any{
				"query":            q,
				"default_field":    spec.Field,
				"analyze_wildcard": true,
			},
		}
	default:
		return map[string]any{"match_phrase": map[string]any{spec.Field: q}}
	}
}

func logStatsSortBody(spec logStatsSpec) []any {
	order := "asc"
	if spec.SortDesc {
		order = "desc"
	}
	return []any{map[string]any{spec.TimeField: map[string]any{"order": order}}}
}

// scanAll 按分片窗口（默认按天）逐段扫描，段内用 scroll 拉完。
// 分片的意义：ES 按天建索引，一次查 2 个月要扫 150 个分片，
// 既容易触发 Kibana 代理 30s 上限，也可能让 ES 超时并静默返回部分结果。
// Kibana 代理超时/失败时，自动改走 ES 直连重试一次。
func (c *logStatsClient) scanAll(ctx context.Context, index string, onDoc func(hit logStatsHit) bool) error {
	return c.scanAllFor(ctx, index, c.spec.Query, onDoc)
}

// scanAllFor 用指定搜索内容扫描（对比项复用同一客户端时用）
func (c *logStatsClient) scanAllFor(ctx context.Context, index, query string, onDoc func(hit logStatsHit) bool) error {
	delivered, err := c.scanChunks(ctx, index, query, onDoc)
	if err == nil {
		return nil
	}
	var httpErr *logStatsHTTPError
	if delivered == 0 && c.endpoint != "es" && errors.As(err, &httpErr) && httpErr.proxyTimeout() {
		if serr := c.switchToES("Kibana 代理超时，已自动改走 ES 直连"); serr == nil {
			_, err2 := c.scanChunks(ctx, index, query, onDoc)
			return err2
		}
	}
	return err
}

// scanChunks 把 [From, To] 切成若干窗口逐段扫描
func (c *logStatsClient) scanChunks(ctx context.Context, index, query string, onDoc func(hit logStatsHit) bool) (int, error) {
	delivered := 0
	for _, win := range logStatsWindows(c.spec.From, c.spec.To, c.spec.Chunk) {
		n, err := c.scanOnce(ctx, index, query, win[0], win[1], onDoc)
		delivered += n
		if err != nil {
			return delivered, err
		}
		if err := ctx.Err(); err != nil {
			return delivered, err
		}
	}
	return delivered, nil
}

// logStatsWindows 按 chunk 切分时间范围；chunk<=0 或范围较小时返回单窗口
func logStatsWindows(from, to time.Time, chunk time.Duration) [][2]time.Time {
	if !to.After(from) {
		return [][2]time.Time{{from, to}}
	}
	if chunk <= 0 {
		return [][2]time.Time{{from, to}}
	}
	var out [][2]time.Time
	cur := from
	for cur.Before(to) {
		end := cur.Add(chunk)
		if end.After(to) {
			end = to
		}
		out = append(out, [2]time.Time{cur, end})
		cur = end
	}
	if len(out) == 0 {
		out = append(out, [2]time.Time{from, to})
	}
	return out
}

// scanOnce 单次完整扫描一个时间窗，返回已回调的条数。
//
// 不用 scroll：ES 的 search.max_open_scroll_context 默认只有 500，
// 大量分页/中断会留下 scroll 上下文，后续查询会静默丢分片（返回 0 条或部分结果）。
// 改用 search_after + 时间戳 tiebreaker 分页，无状态、可中断、不占上下文。
func (c *logStatsClient) scanOnce(ctx context.Context, index, query string, from, to time.Time, onDoc func(hit logStatsHit) bool) (int, error) {
	seen := 0
	var searchAfter []any
	sortBody := logStatsSortWithTiebreaker(c.spec)

	for {
		body := map[string]any{
			"size":             c.spec.PageSize,
			"track_total_hits": false, // 不再依赖 total，避免大范围统计开销
			"_source":          []string{c.spec.TimeField, c.spec.Field},
			"query":            c.buildQueryForQuery(from, to, query),
			"sort":             sortBody,
		}
		if len(searchAfter) > 0 {
			body["search_after"] = searchAfter
		}

		data, status, err := c.call(ctx, http.MethodPost, "/"+index+"/_search", body)
		if err != nil {
			return seen, &logStatsHTTPError{Endpoint: c.endpoint, Cause: err}
		}
		if status >= 300 {
			return seen, &logStatsHTTPError{Endpoint: c.endpoint, Status: status, Body: string(data)}
		}

		page, derr := decodeLogStatsPage(data)
		if derr != nil {
			return seen, derr
		}
		// 分片失败/超时会让结果静默变少，记录下来在汇总里明确提示
		if reason := page.PartialFailure(); reason != "" && c.partialReason == "" {
			c.partialReason = reason
		}
		if page.TimedOut {
			c.partialReason = "ES 查询超时（timed_out）"
		}
		if len(page.Docs) == 0 {
			return seen, nil
		}

		for _, doc := range page.Docs {
			seen++
			if seen > c.spec.MaxDocs {
				c.truncated = true
				return seen, nil
			}
			raw := logStatsDocString(doc.Source, c.spec.Field)
			if raw == "" {
				raw = logStatsDocString(doc.Source, logStatsDefaultField)
			}
			hit := logStatsHit{
				ID:    doc.ID,
				Index: doc.Index,
				TS:    logStatsDocString(doc.Source, c.spec.TimeField),
				Raw:   raw,
			}
			if m := reLogLinePrefix.FindStringSubmatch(raw); m != nil {
				hit.Trace = m[2]
			}
			if !onDoc(hit) {
				return seen, nil
			}
		}

		// 下一页起点：本页最后一条的排序值
		last := page.Docs[len(page.Docs)-1].Sort
		if len(last) == 0 || len(page.Docs) < c.spec.PageSize {
			return seen, nil
		}
		searchAfter = last
		if err := ctx.Err(); err != nil {
			return seen, err
		}
	}
}

// logStatsSortWithTiebreaker 时间字段 + _doc 兜底，保证 search_after 稳定翻页。
// 不用 _id：ES 默认禁用 _id 的 fielddata，排序会直接报错；
// _doc 是 ES 官方推荐的 search_after tiebreaker（无状态、代价最低）。
func logStatsSortWithTiebreaker(spec logStatsSpec) []any {
	order := "asc"
	if spec.SortDesc {
		order = "desc"
	}
	return []any{
		map[string]any{spec.TimeField: map[string]any{"order": order}},
		map[string]any{"_doc": map[string]any{"order": order}},
	}
}

type logStatsPage struct {
	ScrollID string
	Total    int
	TimedOut bool
	Shards   logStatsShards
	Docs     []logStatsDocRef
}

// logStatsShards 分片执行情况：失败分片会让结果静默变少，必须显式报错
type logStatsShards struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"`
	Failures   []struct {
		Index  string `json:"index"`
		Reason struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"reason"`
	} `json:"failures"`
}

// PartialFailure 返回分片失败的简要说明；无失败时为空
func (p logStatsPage) PartialFailure() string {
	if p.Shards.Failed == 0 {
		return ""
	}
	var first string
	for _, f := range p.Shards.Failures {
		if first == "" {
			first = f.Index + ": " + f.Reason.Reason
		}
	}
	if first == "" {
		first = fmt.Sprintf("%d 个分片失败", p.Shards.Failed)
	}
	return first
}

// logStatsDocRef 命中文档的元信息 + 源
type logStatsDocRef struct {
	ID     string
	Index  string
	Source map[string]any
	Sort   []any
}

func decodeLogStatsPage(data []byte) (logStatsPage, error) {
	var out struct {
		ScrollID string         `json:"_scroll_id"`
		TimedOut bool           `json:"timed_out"`
		Shards   logStatsShards `json:"_shards"`
		Hits     struct {
			Total json.RawMessage `json:"total"`
			Hits  []struct {
				ID     string         `json:"_id"`
				Index  string         `json:"_index"`
				Source map[string]any `json:"_source"`
				Sort   []any          `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return logStatsPage{}, fmt.Errorf("解析搜索结果失败: %v（%s）", err, snippet(data))
	}
	page := logStatsPage{ScrollID: out.ScrollID, TimedOut: out.TimedOut, Shards: out.Shards}
	if len(out.Hits.Total) > 0 {
		// ES6 直接给数字，ES7+ 给 {value, relation}
		var n int
		if json.Unmarshal(out.Hits.Total, &n) == nil {
			page.Total = n
		} else {
			var obj struct {
				Value int `json:"value"`
			}
			if json.Unmarshal(out.Hits.Total, &obj) == nil {
				page.Total = obj.Value
			}
		}
	}
	for _, h := range out.Hits.Hits {
		if h.Source != nil {
			page.Docs = append(page.Docs, logStatsDocRef{
				ID: h.ID, Index: h.Index, Source: h.Source, Sort: h.Sort,
			})
		}
	}
	return page, nil
}

func logStatsDocString(doc map[string]any, field string) string {
	if doc == nil || field == "" {
		return ""
	}
	v, ok := doc[field]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// ---------------------------------------------------------------- 输出

func formatLogStatsReport(r *logStatsReport) []string {
	if r == nil {
		return []string{"统计失败：空结果"}
	}
	spec := r.Spec
	lines := []string{
		"──────────────── es 日志统计 ────────────────",
		fmt.Sprintf("请求地址: %s", spec.BaseURL),
		fmt.Sprintf("端点    : %s", r.Endpoint),
		fmt.Sprintf("索引    : %s", r.ResolvedIdx),
		fmt.Sprintf("时间范围: %s ~ %s%s", formatLogStatsBound(spec.From), formatLogStatsBound(spec.To), rangeLabelSuffix(spec)),
		fmt.Sprintf("查询条件: %s %s %q", spec.Field, spec.QueryMode, spec.Query),
		fmt.Sprintf("提取规则: %s", spec.PatternText),
	}
	if strings.TrimSpace(spec.CompareQuery) != "" {
		lines = append(lines,
			fmt.Sprintf("对比条件: %s %s %q", spec.Field, spec.QueryMode, spec.CompareQuery),
			fmt.Sprintf("对比规则: %s", spec.ComparePatternText),
		)
	}
	lines = append(lines,
		fmt.Sprintf("去重    : %s", describeDedupMode(spec.DedupMode)),
		"",
	)

	if !spec.SummaryOnly {
		if len(r.Rows) == 0 {
			lines = append(lines, "（未提取到任何数值）")
		} else {
			for i, row := range r.Rows {
				line := fmt.Sprintf("[%d] %s  %d", i+1, formatLogStatsTimestamp(row.Timestamp), row.Number)
				if spec.ShowRaw {
					line += "  |  " + truncateRunes(row.Raw, 200)
				}
				lines = append(lines, line)
			}
		}
		lines = append(lines, "")
	}

	if spec.ShowDup && len(r.DupSamples) > 0 {
		lines = append(lines, fmt.Sprintf("── 被跳过的重复（最多列 %d 条）──", logStatsDupSampleLimit))
		for i, row := range r.DupSamples {
			line := fmt.Sprintf("[dup %d] %s  %d", i+1, formatLogStatsTimestamp(row.Timestamp), row.Number)
			if row.Index != "" {
				line += "  @" + row.Index
			}
			lines = append(lines, line+"  |  "+truncateRunes(row.Raw, 160))
		}
		lines = append(lines, "")
	}

	if spec.GroupBy != "" && len(r.Groups) > 0 {
		keys := make([]string, 0, len(r.Groups))
		for k := range r.Groups {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		unit := "按小时"
		if strings.EqualFold(spec.GroupBy, "day") {
			unit = "按天"
		}
		lines = append(lines, fmt.Sprintf("── %s汇总 ──", unit))
		for _, k := range keys {
			g := r.Groups[k]
			lines = append(lines, fmt.Sprintf("%-16s 条数 %-4d 合计 %-12d 均值 %.1f",
				k, g.Count, g.Sum, float64(g.Sum)/float64(g.Count)))
		}
		lines = append(lines, "")
	}

	switch {
	case spec.DedupMode == "none":
		lines = append(lines, fmt.Sprintf("命中日志: %d 条（未去重）；可解析出数值: %d 条", r.Docs, r.Matched))
	case r.DupSkipped > 0:
		lines = append(lines, fmt.Sprintf("命中日志: %d 条（去重后；原始 %d 条，跳过 %d 条重复）；可解析出数值: %d 条",
			r.Docs, r.Docs+r.DupSkipped, r.DupSkipped, r.Matched))
	default:
		lines = append(lines, fmt.Sprintf("命中日志: %d 条（已去重，无重复）；可解析出数值: %d 条", r.Docs, r.Matched))
	}
	if r.HasNumber {
		lines = append(lines,
			fmt.Sprintf("合计总条数: %d", r.Sum),
			fmt.Sprintf("单条最小/最大: %d / %d，平均 %.1f", r.Min, r.Max, float64(r.Sum)/float64(r.Matched)),
		)
		if r.FirstTS != "" || r.LastTS != "" {
			lines = append(lines, fmt.Sprintf("覆盖时间: %s ~ %s",
				formatLogStatsTimestamp(r.FirstTS), formatLogStatsTimestamp(r.LastTS)))
		}
	} else {
		lines = append(lines, "合计总条数: 0（没有解析到数值，检查 --query / --pattern）")
	}
	// 批量 INSERT 汇总（按金额/币种）
	if r.SQL != nil {
		lines = append(lines, "")
		lines = append(lines, "── SQL 批量插入汇总 ──")
		lines = append(lines, formatSQLAgg(r.SQL, r.SQL.TradeCol)...)
	}

	// 对比项 + 差额
	if r.Compare != nil {
		cmp := r.Compare
		lines = append(lines,
			"",
			"── 对比统计 ──",
			fmt.Sprintf("%-14s %-10s 条数 %-5d 合计 %d",
				"主项", truncateRunes(spec.Query, 14), r.Matched, r.Sum),
			fmt.Sprintf("%-14s %-10s 条数 %-5d 合计 %d",
				"对比项", truncateRunes(spec.CompareQuery, 14), cmp.Matched, cmp.Sum),
		)
		diff := buildDiff(r)
		if diff != nil {
			sign := ""
			if diff.Diff() > 0 {
				sign = "（主项多）"
			} else if diff.Diff() < 0 {
				sign = "（主项少）"
			}
			lines = append(lines,
				fmt.Sprintf("差额    : %d  %s", diff.Diff(), sign),
				fmt.Sprintf("占比    : 对比项为主项的 %.2f%%", diff.Ratio()),
			)
		}

		// 逐日对比（两边都按天分组时才有意义）
		if spec.GroupBy != "" && len(r.Groups) > 0 {
			days := make([]string, 0, len(r.Groups))
			for k := range r.Groups {
				days = append(days, k)
			}
			sort.Strings(days)
			lines = append(lines,
				"",
				"── 逐日对比（主项 - 对比项）──",
				fmt.Sprintf("%-16s %12s %12s %12s", "日期", "数据条数", "退单条数", "差额"),
			)
			var totalMain, totalCompare int64
			for _, d := range days {
				mg := r.Groups[d]
				var compareSum int64
				if cmp.Groups != nil {
					if cg := cmp.Groups[d]; cg != nil {
						compareSum = cg.Sum
					}
				}
				totalMain += mg.Sum
				totalCompare += compareSum
				lines = append(lines, fmt.Sprintf("%-16s %12d %12d %12d", d, mg.Sum, compareSum, mg.Sum-compareSum))
			}
			lines = append(lines,
				fmt.Sprintf("%-16s %12d %12d %12d", "合计", totalMain, totalCompare, totalMain-totalCompare),
			)
		}
		if cmp.Partial != "" {
			lines = append(lines, fmt.Sprintf("注意: 对比项部分分片未返回完整结果（%s）", cmp.Partial))
		}
		lines = append(lines, "")
	}

	if r.Truncated {
		lines = append(lines, fmt.Sprintf("注意: 命中数超过 --max-docs=%d，只统计了前 %d 条", spec.MaxDocs, spec.MaxDocs))
	}
	if r.Partial != "" {
		lines = append(lines, fmt.Sprintf("注意: 部分分片未返回完整结果（%s），合计可能偏小；可加大 --timeout 或调小 --chunk", r.Partial))
	}
	return lines
}

// describeDedupMode 把去重模式翻成说明
func describeDedupMode(mode string) string {
	switch mode {
	case "none":
		return "关闭（--dedup none）"
	case "id":
		return "按索引+_id（只去同一文档重复命中）"
	case "timestamp", "ts":
		return "按时间戳+提取值"
	default:
		return "按日志正文（默认，重复采集的同一条只算一次；Kibana 页面看到的是未去重条数）"
	}
}

// rangeLabelSuffix 展示 --range 预设名，方便核对是哪档区间
func rangeLabelSuffix(spec logStatsSpec) string {
	if spec.RangeLabel == "" {
		return ""
	}
	return "（" + spec.RangeLabel + "）"
}

// formatLogStatsBound 边界时间带毫秒，便于核对 .999 / .000
func formatLogStatsBound(t time.Time) string {
	t = t.Local()
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02 15:04:05")
	}
	return t.Format("2006-01-02 15:04:05.999")
}

func snippet(data []byte) string {
	s := strings.TrimSpace(string(data))
	s = strings.Join(strings.Fields(s), " ")
	return truncateRunes(s, 300)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
