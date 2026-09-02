package mcp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"FlashDock/crypto"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Token 作用域令牌（落盘/API 列表/编辑）。只存 SHA256，不含明文、不可恢复。
type Token struct {
	ID        string   `yaml:"id" json:"id"`
	Name      string   `yaml:"name" json:"name"`
	Client    string   `yaml:"client" json:"client"`
	Hash      string   `yaml:"hash,omitempty" json:"-"`
	Secret    string   `yaml:"secret,omitempty" json:"-"` // 旧版可逆密文，加载时迁移为 Hash 后清空
	Servers   []string `yaml:"servers,omitempty" json:"servers"`
	CIDRs     []string `yaml:"cidrs,omitempty" json:"cidrs"`
	CreatedAt string   `yaml:"createdAt" json:"createdAt"`
	LastUsed  string   `yaml:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
}

// IssuedToken 签发响应：Plaintext 仅此一次经 API 返回，之后任何接口均不可恢复。
type IssuedToken struct {
	Token
	Plaintext string `json:"token"`
}

// IssueOpts 签发 / 一键接入参数
type IssueOpts struct {
	Name    string   `json:"name"`
	Client  string   `json:"client"`
	Servers []string `json:"servers"` // 空 = 全部可见
	CIDRs   []string `json:"cidrs"`   // 空 = 127.0.0.1/32
}

// UpdateTokenOpts 更新已有 token 的作用域（不轮换明文）
type UpdateTokenOpts struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Servers []string `json:"servers"` // 空 = 全部可见
	CIDRs   []string `json:"cidrs"`   // 空 = 127.0.0.1/32
}

type tokenFile struct {
	Tokens []Token `yaml:"tokens"`
}

type TokenStore struct {
	mu       sync.Mutex
	tokens   []Token
	fileMod  time.Time // tokens.yaml mtime（辅助）
	fileHash string    // tokens.yaml 内容 SHA256，用于跨进程同步
}

func tokensFileDigest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func tokensFilePath() (string, error) {
	root, err := homeDir()
	if err != nil {
		return "", err
	}
	return join(root, tokensFile), nil
}

func loadTokens() *TokenStore {
	st := &TokenStore{}
	_ = st.reloadLocked()
	return st
}

// reloadLocked 从磁盘重载（调用方须已持锁）。文件不存在时保持空列表。
func (st *TokenStore) reloadLocked() error {
	path, err := tokensFilePath()
	if err != nil {
		return err
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			st.tokens = nil
			st.fileMod = time.Time{}
			st.fileHash = ""
			return nil
		}
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f tokenFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}
	changed := false
	for i := range f.Tokens {
		t := &f.Tokens[i]
		if t.Hash == "" && t.Secret != "" {
			if p, err := crypto.DecryptText(t.Secret); err == nil && p != "" {
				t.Hash = hashToken(p)
				t.Secret = ""
				changed = true
			}
		}
		if len(t.CIDRs) == 0 {
			t.CIDRs = []string{"127.0.0.1/32"}
		}
	}
	st.tokens = f.Tokens
	st.fileMod = fi.ModTime()
	st.fileHash = tokensFileDigest(b)
	if changed {
		return st.saveLocked()
	}
	return nil
}

// syncFromDiskLocked 若磁盘内容与内存不一致则重载，避免 MCP 写 lastUsedAt 时用旧作用域覆盖 UI 编辑。
func (st *TokenStore) syncFromDiskLocked() {
	path, err := tokensFilePath()
	if err != nil {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if st.fileHash != "" || len(st.tokens) > 0 {
				st.tokens = nil
				st.fileHash = ""
				st.fileMod = time.Time{}
			}
		}
		return
	}
	if tokensFileDigest(b) != st.fileHash {
		_ = st.reloadLocked()
	}
}

func (st *TokenStore) save() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.saveLocked()
}

func (st *TokenStore) saveLocked() error {
	path, err := tokensFilePath()
	if err != nil {
		return err
	}
	out := make([]Token, len(st.tokens))
	copy(out, st.tokens)
	for i := range out {
		out[i].Secret = ""
	}
	b, err := yaml.Marshal(tokenFile{Tokens: out})
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return err
	}
	st.fileHash = tokensFileDigest(b)
	if fi, err := os.Stat(path); err == nil {
		st.fileMod = fi.ModTime()
	} else {
		st.fileMod = time.Now()
	}
	return nil
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

func decodeYAMLTime(n *yaml.Node) string {
	if n == nil || n.Kind == 0 {
		return ""
	}
	var ts time.Time
	if err := n.Decode(&ts); err == nil && !ts.IsZero() {
		return ts.Format(time.RFC3339)
	}
	return strings.TrimSpace(n.Value)
}

// UnmarshalYAML 兼容旧盘 YAML 把 createdAt 写成 timestamp 的情况。
func (t *Token) UnmarshalYAML(value *yaml.Node) error {
	type tokenYAML struct {
		ID        string    `yaml:"id"`
		Name      string    `yaml:"name"`
		Client    string    `yaml:"client"`
		Hash      string    `yaml:"hash,omitempty"`
		Secret    string    `yaml:"secret,omitempty"`
		Servers   []string  `yaml:"servers,omitempty"`
		CIDRs     []string  `yaml:"cidrs,omitempty"`
		CreatedAt yaml.Node `yaml:"createdAt"`
		LastUsed  yaml.Node `yaml:"lastUsedAt,omitempty"`
	}
	var raw tokenYAML
	if err := value.Decode(&raw); err != nil {
		return err
	}
	t.ID = raw.ID
	t.Name = raw.Name
	t.Client = raw.Client
	t.Hash = raw.Hash
	t.Secret = raw.Secret
	t.Servers = raw.Servers
	t.CIDRs = raw.CIDRs
	t.CreatedAt = decodeYAMLTime(&raw.CreatedAt)
	t.LastUsed = decodeYAMLTime(&raw.LastUsed)
	return nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (st *TokenStore) List() []Token {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	out := make([]Token, len(st.tokens))
	for i, t := range st.tokens {
		t.Hash = ""
		t.Secret = ""
		out[i] = t
	}
	return out
}

func (st *TokenStore) Get(id string) (Token, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	for _, t := range st.tokens {
		if t.ID == id {
			t.Hash = ""
			t.Secret = ""
			return t, true
		}
	}
	return Token{}, false
}

func normalizeIssue(opts IssueOpts) (IssueOpts, error) {
	opts.Name = strings.TrimSpace(opts.Name)
	opts.Client = strings.TrimSpace(opts.Client)
	if opts.Name == "" {
		opts.Name = "Token"
	}
	if opts.Client == "" {
		opts.Client = "manual"
	}
	var servers []string
	for _, s := range opts.Servers {
		s = strings.TrimSpace(s)
		if s != "" {
			servers = append(servers, s)
		}
	}
	opts.Servers = servers
	cidrs := opts.CIDRs
	if len(cidrs) == 0 {
		cidrs = []string{"127.0.0.1/32"}
	}
	norm, err := normalizeCIDRs(cidrs)
	if err != nil {
		return opts, err
	}
	opts.CIDRs = norm
	return opts, nil
}

func normalizeCIDRs(cidrs []string) ([]string, error) {
	var out []string
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			if ip := net.ParseIP(c); ip != nil {
				if ip.To4() != nil {
					c = c + "/32"
				} else {
					c = c + "/128"
				}
			}
		}
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			return nil, fmt.Errorf("无效 CIDR: %s", c)
		}
		if isPublicCIDR(n) {
			return nil, fmt.Errorf("不允许公网 CIDR: %s（仅本机/私网）", c)
		}
		out = append(out, n.String())
	}
	if len(out) == 0 {
		out = []string{"127.0.0.1/32"}
	}
	return out, nil
}

func isPublicCIDR(n *net.IPNet) bool {
	ones, bits := n.Mask.Size()
	if bits == 0 {
		return true
	}
	if ones < 8 {
		return true
	}
	ip := n.IP
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return false
	}
	return true
}

func ipAllowed(ipStr string, cidrs []string) bool {
	host := ipStr
	if h, _, err := net.SplitHostPort(ipStr); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if len(cidrs) == 0 {
		cidrs = []string{"127.0.0.1/32"}
	}
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			continue
		}
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (st *TokenStore) Issue(opts IssueOpts) (IssuedToken, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	opts, err := normalizeIssue(opts)
	if err != nil {
		return IssuedToken{}, err
	}
	rec, plain, err := st.issueLocked(opts)
	if err != nil {
		return IssuedToken{}, err
	}
	return IssuedToken{Token: rec, Plaintext: plain}, nil
}

func (st *TokenStore) issueLocked(opts IssueOpts) (Token, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(raw)
	rec := Token{
		ID:        "tok_" + uuid.NewString()[:8],
		Name:      opts.Name,
		Client:    opts.Client,
		Hash:      hashToken(plain),
		Servers:   append([]string{}, opts.Servers...),
		CIDRs:     append([]string{}, opts.CIDRs...),
		CreatedAt: nowRFC3339(),
	}
	st.tokens = append(st.tokens, rec)
	if err := st.saveLocked(); err != nil {
		return Token{}, "", err
	}
	return rec, plain, nil
}

// EnsureClient 若该 client 已有 token 则返回；否则内部签发（明文不返回，须走一键接入或刷新）。
func (st *TokenStore) EnsureClient(name, client string) (Token, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	for _, t := range st.tokens {
		if t.Client == client && t.Hash != "" {
			t.Hash = ""
			t.Secret = ""
			return t, nil
		}
	}
	opts, err := normalizeIssue(IssueOpts{Name: name, Client: client})
	if err != nil {
		return Token{}, err
	}
	rec, _, err := st.issueLocked(opts)
	return rec, err
}

// IssueForClient 为客户端重新签发（替换同 client 旧 token），返回明文一次。
func (st *TokenStore) IssueForClient(opts IssueOpts) (IssuedToken, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	opts, err := normalizeIssue(opts)
	if err != nil {
		return IssuedToken{}, err
	}
	next := st.tokens[:0]
	for _, t := range st.tokens {
		if strings.EqualFold(t.Client, opts.Client) {
			continue
		}
		next = append(next, t)
	}
	st.tokens = next
	rec, plain, err := st.issueLocked(opts)
	if err != nil {
		return IssuedToken{}, err
	}
	return IssuedToken{Token: rec, Plaintext: plain}, nil
}

func (st *TokenStore) Generate(name, client string) (IssuedToken, error) {
	return st.Issue(IssueOpts{Name: name, Client: client})
}

// Update 更新名称 / 可见服务器 / CIDR，不轮换 hash（客户端无需重新粘贴 Token）
func (st *TokenStore) Update(opts UpdateTokenOpts) (Token, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	id := strings.TrimSpace(opts.ID)
	if id == "" {
		return Token{}, fmt.Errorf("token id 为空")
	}
	norm, err := normalizeIssue(IssueOpts{
		Name:    opts.Name,
		Servers: opts.Servers,
		CIDRs:   opts.CIDRs,
	})
	if err != nil {
		return Token{}, err
	}
	for i := range st.tokens {
		if st.tokens[i].ID != id {
			continue
		}
		if strings.TrimSpace(opts.Name) != "" {
			st.tokens[i].Name = norm.Name
		}
		st.tokens[i].Servers = append([]string{}, norm.Servers...)
		st.tokens[i].CIDRs = append([]string{}, norm.CIDRs...)
		out := st.tokens[i]
		out.Hash = ""
		out.Secret = ""
		if err := st.saveLocked(); err != nil {
			return Token{}, err
		}
		return out, nil
	}
	return Token{}, fmt.Errorf("token 不存在")
}

func (st *TokenStore) Revoke(id string) (Token, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	var removed Token
	next := st.tokens[:0]
	for _, t := range st.tokens {
		if t.ID == id {
			removed = t
			continue
		}
		next = append(next, t)
	}
	if removed.ID == "" {
		return Token{}, fmt.Errorf("token 不存在")
	}
	st.tokens = next
	return removed, st.saveLocked()
}

func (st *TokenStore) RevokeByClient(client string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.syncFromDiskLocked()
	client = strings.ToLower(strings.TrimSpace(client))
	next := st.tokens[:0]
	for _, t := range st.tokens {
		if strings.ToLower(t.Client) != client {
			next = append(next, t)
		}
	}
	st.tokens = next
	return st.saveLocked()
}

func (st *TokenStore) Clear() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.tokens = nil
	return st.saveLocked()
}

func (st *TokenStore) Valid(plain string) (Token, bool) {
	return st.ValidFrom(plain, "127.0.0.1")
}

func (st *TokenStore) ValidFrom(plain, remoteIP string) (Token, bool) {
	if plain == "" {
		return Token{}, false
	}
	h := hashToken(plain)
	st.mu.Lock()
	defer st.mu.Unlock()
	// 先跟磁盘对齐：UI 改完 servers 后，stdio 进程必须用最新作用域，
	// 否则写 lastUsedAt 时会把旧 servers 覆盖回 tokens.yaml。
	st.syncFromDiskLocked()
	for i, t := range st.tokens {
		if t.Hash == "" || t.Hash != h {
			continue
		}
		if !ipAllowed(remoteIP, t.CIDRs) {
			return Token{}, false
		}
		out, err := st.markTokenUsedLocked(st.tokens[i].ID)
		if err != nil {
			return Token{}, false
		}
		return out, true
	}
	return Token{}, false
}

// markTokenUsedLocked 写 lastUsedAt 前先跟磁盘对齐，只改时间戳，避免覆盖 servers/cidrs。
func (st *TokenStore) markTokenUsedLocked(id string) (Token, error) {
	lastUsed := nowRFC3339()
	st.syncFromDiskLocked()
	for i := range st.tokens {
		if st.tokens[i].ID != id {
			continue
		}
		st.tokens[i].LastUsed = lastUsed
		if err := st.saveLocked(); err != nil {
			return Token{}, err
		}
		out := st.tokens[i]
		out.Hash = ""
		out.Secret = ""
		return out, nil
	}
	return Token{}, fmt.Errorf("token 不存在")
}

func (st *TokenStore) First() (Token, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.tokens) == 0 {
		return Token{}, false
	}
	t := st.tokens[0]
	t.Hash = ""
	t.Secret = ""
	return t, true
}

// SeesServer 空 Servers = 全部可见
func (t Token) SeesServer(alias string) bool {
	if len(t.Servers) == 0 {
		return true
	}
	alias = strings.TrimSpace(alias)
	for _, s := range t.Servers {
		if strings.EqualFold(strings.TrimSpace(s), alias) {
			return true
		}
	}
	return false
}
