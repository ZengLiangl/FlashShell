package netproxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// Mode / Type 常量
const (
	ModeNone   = "none"
	ModeManual = "manual"
	TypeHTTP   = "http"
	TypeSOCKS  = "socks"
)

// Settings 运行时代理配置（与 data.ProxySettings 对应）
type Settings struct {
	Mode     string
	Type     string
	Host     string
	Port     int
	User     string
	Password string
}

var (
	mu        sync.RWMutex
	current   = Settings{Mode: ModeNone, Type: TypeHTTP}
	transport = newDirectTransport()
)

// Normalize 规范化配置
func Normalize(s Settings) Settings {
	mode := strings.ToLower(strings.TrimSpace(s.Mode))
	if mode != ModeManual {
		mode = ModeNone
	}
	typ := strings.ToLower(strings.TrimSpace(s.Type))
	if typ != TypeSOCKS {
		typ = TypeHTTP
	}
	host := strings.TrimSpace(s.Host)
	port := s.Port
	if port < 0 || port > 65535 {
		port = 0
	}
	user := strings.TrimSpace(s.User)
	// 密码保留原样（可能含首尾空格）；仅去掉纯空白
	password := s.Password
	if strings.TrimSpace(password) == "" {
		password = ""
	}
	if mode == ModeManual && (host == "" || port == 0) {
		mode = ModeNone
	}
	if mode != ModeManual {
		user = ""
		password = ""
	}
	return Settings{Mode: mode, Type: typ, Host: host, Port: port, User: user, Password: password}
}

// Apply 应用代理到全局 HTTP Transport 与返回的 DialContext
func Apply(s Settings) {
	s = Normalize(s)
	t := buildTransport(s)
	mu.Lock()
	current = s
	transport = t
	mu.Unlock()
}

// Current 返回当前已应用配置的副本
func Current() Settings {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// HTTPTransport 返回当前 HTTP Transport（只读使用，勿修改）
func HTTPTransport() *http.Transport {
	mu.RLock()
	defer mu.RUnlock()
	return transport
}

// HTTPClient 使用当前代理配置创建客户端
func HTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: HTTPTransport(),
	}
}

// DialContext 按当前配置拨号（SSH 等用）
func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	mu.RLock()
	s := current
	mu.RUnlock()
	return dialWith(ctx, s, network, address)
}

// Test 用指定配置（可不先 Apply）请求 testURL，检测连通性
func Test(ctx context.Context, s Settings, testURL string) error {
	s = Normalize(s)
	testURL = strings.TrimSpace(testURL)
	if testURL == "" {
		return fmt.Errorf("请输入测试地址")
	}
	if !strings.Contains(testURL, "://") {
		testURL = "https://" + testURL
	}
	u, err := url.Parse(testURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("测试地址无效")
	}

	tr := buildTransport(s)
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: tr,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "FlashDock-ProxyCheck/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("服务器错误: HTTP %d", resp.StatusCode)
	}
	return nil
}

func proxyAuth(s Settings) *proxy.Auth {
	if s.User == "" && s.Password == "" {
		return nil
	}
	return &proxy.Auth{User: s.User, Password: s.Password}
}

func dialWith(ctx context.Context, s Settings, network, address string) (net.Conn, error) {
	if network == "" {
		network = "tcp"
	}
	direct := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if s.Mode != ModeManual {
		return direct.DialContext(ctx, network, address)
	}
	proxyAddr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	switch s.Type {
	case TypeSOCKS:
		socksDialer, err := proxy.SOCKS5("tcp", proxyAddr, proxyAuth(s), direct)
		if err != nil {
			return nil, fmt.Errorf("初始化 SOCKS 代理失败: %w", err)
		}
		if ctxDialer, ok := socksDialer.(proxy.ContextDialer); ok {
			return ctxDialer.DialContext(ctx, network, address)
		}
		type dialResult struct {
			conn net.Conn
			err  error
		}
		ch := make(chan dialResult, 1)
		go func() {
			c, e := socksDialer.Dial(network, address)
			ch <- dialResult{c, e}
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-ch:
			return r.conn, r.err
		}
	default:
		return dialHTTPConnect(ctx, direct, proxyAddr, address, s.User, s.Password)
	}
}

func dialHTTPConnect(ctx context.Context, direct *net.Dialer, proxyAddr, target, user, password string) (net.Conn, error) {
	conn, err := direct.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("连接 HTTP 代理失败: %w", err)
	}

	ok := false
	defer func() {
		if !ok {
			_ = conn.Close()
		}
	}()

	if deadline, okd := ctx.Deadline(); okd {
		_ = conn.SetDeadline(deadline)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n", target, target)
	if user != "" || password != "" {
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
		fmt.Fprintf(&b, "Proxy-Authorization: Basic %s\r\n", token)
	}
	b.WriteString("Proxy-Connection: Keep-Alive\r\n\r\n")
	if _, err := io.WriteString(conn, b.String()); err != nil {
		return nil, fmt.Errorf("代理 CONNECT 请求失败: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		return nil, fmt.Errorf("读取代理响应失败: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("代理 CONNECT 被拒绝: %s", resp.Status)
	}
	_ = conn.SetDeadline(time.Time{})

	if br.Buffered() > 0 {
		ok = true
		return &bufferedConn{Conn: conn, r: br}, nil
	}
	ok = true
	return conn, nil
}

type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func newDirectTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	}
}

func buildTransport(s Settings) *http.Transport {
	s = Normalize(s)
	tr := &http.Transport{
		MaxIdleConns:          32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DisableCompression:    true,
		WriteBufferSize:       512 * 1024,
		ReadBufferSize:        512 * 1024,
	}
	if s.Mode != ModeManual {
		tr.Proxy = http.ProxyFromEnvironment
		tr.DialContext = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext
		return tr
	}

	proxyAddr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	switch s.Type {
	case TypeSOCKS:
		tr.Proxy = nil
		tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialWith(ctx, s, network, address)
		}
	default:
		u := &url.URL{
			Scheme: "http",
			Host:   proxyAddr,
		}
		if s.User != "" || s.Password != "" {
			u.User = url.UserPassword(s.User, s.Password)
		}
		tr.Proxy = http.ProxyURL(u)
		tr.DialContext = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	return tr
}

// ProxyURLString 调试用（不含鉴权）
func ProxyURLString(s Settings) string {
	s = Normalize(s)
	if s.Mode != ModeManual {
		return ""
	}
	scheme := "http"
	if s.Type == TypeSOCKS {
		scheme = "socks5"
	}
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
}
