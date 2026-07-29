package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/netproxy"
)

func proxySettingsFromData(p data.ProxySettings) netproxy.Settings {
	password, _ := p.GetProxyPassword()
	return netproxy.Settings{
		Mode:     p.Mode,
		Type:     p.Type,
		Host:     p.Host,
		Port:     p.Port,
		User:     strings.TrimSpace(p.User),
		Password: password,
	}
}

// normalizeProxySettings 规范化展示/运行时字段；不改写已落盘的加密密码。
func normalizeProxySettings(p *data.ProxySettings) {
	if p == nil {
		return
	}
	p.User = strings.TrimSpace(p.User)
	n := netproxy.Normalize(proxySettingsFromData(*p))
	p.Mode = n.Mode
	p.Type = n.Type
	p.Host = n.Host
	p.Port = n.Port
	p.User = n.User
	if p.Port == 0 {
		p.Port = 7890
	}
}

// prepareProxySettingsForSave 保存前加密密码，并在关闭代理时清掉认证。
func prepareProxySettingsForSave(p *data.ProxySettings) error {
	if p == nil {
		return nil
	}
	p.User = strings.TrimSpace(p.User)
	if err := p.PrepareProxyPasswordForSave(); err != nil {
		return err
	}
	normalizeProxySettings(p)
	if p.Mode != netproxy.ModeManual {
		p.User = ""
		return p.SetProxyPassword("")
	}
	return nil
}

func (a *App) applyProxySettings(p data.ProxySettings) {
	s := netproxy.Normalize(proxySettingsFromData(p))
	netproxy.Apply(s)
	define.SetDialContext(netproxy.DialContext)
}

func (a *App) applyProxyFromConfig() {
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil || cfg == nil {
		a.applyProxySettings(data.ProxySettings{Mode: netproxy.ModeNone, Type: netproxy.TypeHTTP, Port: 7890})
		return
	}
	a.applyProxySettings(cfg.ProxySettings)
}

// TestProxyConnection 用给定代理配置测试访问 testURL（可不先保存）
func (a *App) TestProxyConnection(settings data.ProxySettings, testURL string) (string, error) {
	testURL = strings.TrimSpace(testURL)
	if testURL == "" {
		return "", fmt.Errorf("请输入测试地址")
	}
	settings.User = strings.TrimSpace(settings.User)
	normalizeProxySettings(&settings)
	if settings.Mode == netproxy.ModeManual && (strings.TrimSpace(settings.Host) == "" || settings.Port <= 0) {
		return "", fmt.Errorf("请填写代理主机和端口")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := netproxy.Test(ctx, proxySettingsFromData(settings), testURL); err != nil {
		return "", err
	}
	return "连接成功", nil
}
