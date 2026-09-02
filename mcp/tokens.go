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

// Token 作用域令牌。落盘只存 SHA256；明文仅在签发当次返回。
type Token struct {
	ID        string   `yaml:"id" json:"id"`
	Name      string   `yaml:"name" json:"name"`
	Client    string   `yaml:"client" json:"client"`
	Hash      string   `yaml:"hash,omitempty" json:"-"`
	Secret    string   `yaml:"secret,omitempty" json:"-"` // 旧版可逆密文，迁移后清空
	Plain     string   `yaml:"-" json:"token,omitempty"`  // 仅签发响应
	Servers   []string `yaml:"servers,omitempty" json:"servers"`
	CIDRs     []string `yaml:"cidrs,omitempty" json:"cidrs"`
	CreatedAt string   `yaml:"createdAt" json:"createdAt"`
	LastUsed  string   `yaml:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
}

// IssueOpts 签发 / 一键接入参数
type IssueOpts struct {
	Name    string   `json:"name"`
	Client  string   `json:"client"`
	Servers []string `json:"servers"` // 空 = 全部可见
	CIDRs   []string `json:"cidrs"`   // 空 = 127.0.0.1/32
}

type tokenFile struct {
	Tokens []Token `yaml:"tokens"`
}

type TokenStore struct {
	mu     sync.Mutex
	tokens []Token
}

func loadTokens() *TokenStore {
	st := &TokenStore{}
	root, err := homeDir()
	if err != nil {
		return st
	}
	b, err := os.ReadFile(join(root, tokensFile))
	if err != nil {
		return st
	}
	var f tokenFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return st
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
		t.Plain = ""
	}
	st.tokens = f.Tokens
	if changed {
		_ = st.save()
	}
	return st
}

func (st *TokenStore) save() error {
	root, err := homeDir()
	if err != nil {
		return err
	}
	out := make([]Token, len(st.tokens))
	copy(out, st.tokens)
	for i := range out {
		out[i].Plain = ""
		out[i].Secret = ""
	}
	b, err := yaml.Marshal(tokenFile{Tokens: out})
	if err != nil {
		return err
	}
	return os.WriteFile(join(root, tokensFile), b, 0600)
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
	out := make([]Token, len(st.tokens))
	for i, t := range st.tokens {
		t.Plain = ""
		t.Hash = ""
		t.Secret = ""
		out[i] = t
	}
	return out
}

func (st *TokenStore) Get(id string) (Token, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, t := range st.tokens {
		if t.ID == id {
			t.Plain = ""
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

func (st *TokenStore) Issue(opts IssueOpts) (Token, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	opts, err := normalizeIssue(opts)
	if err != nil {
		return Token{}, err
	}
	return st.issueLocked(opts)
}

func (st *TokenStore) issueLocked(opts IssueOpts) (Token, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, err
	}
	plain := base64.RawURLEncoding.EncodeToString(raw)
	t := Token{
		ID:        "tok_" + uuid.NewString()[:8],
		Name:      opts.Name,
		Client:    opts.Client,
		Hash:      hashToken(plain),
		Plain:     plain,
		Servers:   append([]string{}, opts.Servers...),
		CIDRs:     append([]string{}, opts.CIDRs...),
		CreatedAt: nowRFC3339(),
	}
	st.tokens = append(st.tokens, Token{
		ID: t.ID, Name: t.Name, Client: t.Client, Hash: t.Hash,
		Servers: t.Servers, CIDRs: t.CIDRs, CreatedAt: t.CreatedAt,
	})
	if err := st.save(); err != nil {
		return Token{}, err
	}
	return t, nil
}

// EnsureClient 若该 client 已有 token 则返回（无明文）；否则签发新的（带明文一次）。
func (st *TokenStore) EnsureClient(name, client string) (Token, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, t := range st.tokens {
		if t.Client == client && t.Hash != "" {
			t.Plain = ""
			return t, nil
		}
	}
	opts, err := normalizeIssue(IssueOpts{Name: name, Client: client})
	if err != nil {
		return Token{}, err
	}
	return st.issueLocked(opts)
}

// IssueForClient 为客户端重新签发（替换同 client 旧 token），返回明文一次。
func (st *TokenStore) IssueForClient(opts IssueOpts) (Token, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	opts, err := normalizeIssue(opts)
	if err != nil {
		return Token{}, err
	}
	next := st.tokens[:0]
	for _, t := range st.tokens {
		if strings.EqualFold(t.Client, opts.Client) {
			continue
		}
		next = append(next, t)
	}
	st.tokens = next
	return st.issueLocked(opts)
}

func (st *TokenStore) Generate(name, client string) (Token, error) {
	return st.Issue(IssueOpts{Name: name, Client: client})
}

func (st *TokenStore) Revoke(id string) (Token, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
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
	return removed, st.save()
}

func (st *TokenStore) RevokeByClient(client string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	client = strings.ToLower(strings.TrimSpace(client))
	next := st.tokens[:0]
	for _, t := range st.tokens {
		if strings.ToLower(t.Client) != client {
			next = append(next, t)
		}
	}
	st.tokens = next
	return st.save()
}

func (st *TokenStore) Clear() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.tokens = nil
	return st.save()
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
	for i, t := range st.tokens {
		ok := t.Hash != "" && t.Hash == h
		if !ok && t.Secret != "" {
			if p, err := crypto.DecryptText(t.Secret); err == nil && p == plain {
				st.tokens[i].Hash = h
				st.tokens[i].Secret = ""
				ok = true
				_ = st.save()
			}
		}
		if !ok {
			continue
		}
		if !ipAllowed(remoteIP, t.CIDRs) {
			return Token{}, false
		}
		st.tokens[i].LastUsed = nowRFC3339()
		_ = st.save()
		out := st.tokens[i]
		out.Plain = ""
		out.Hash = ""
		out.Secret = ""
		return out, true
	}
	return Token{}, false
}

func (st *TokenStore) First() (Token, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.tokens) == 0 {
		return Token{}, false
	}
	t := st.tokens[0]
	t.Plain = ""
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
