package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) handleWebStatus(_ context.Context, a ServerOnly) (any, error) {
	cmd := `sh -c 'if docker ps --format "{{.Names}}" 2>/dev/null | grep -Eq "^(flashshell-openresty|reeve-openresty)$"; then echo "mode=docker running=yes"; docker ps --filter name=openresty --format "{{.Status}}"; elif command -v openresty >/dev/null; then echo "mode=native"; openresty -v 2>&1; elif command -v nginx >/dev/null; then echo "mode=nginx"; nginx -v 2>&1; else echo "mode=none running=no"; fi'`
	return s.execSSH(a.Server, cmd, 20*time.Second)
}

func (s *Service) handleWebInstall(_ context.Context, a ServerOnly) (any, error) {
	script := `set -e
mkdir -p /opt/flashshell/openresty/conf /opt/flashshell/openresty/html /opt/flashshell/openresty/certs
if ! command -v docker >/dev/null; then echo "需要 Docker"; exit 1; fi
docker rm -f flashshell-openresty 2>/dev/null || true
docker run -d --name flashshell-openresty --restart unless-stopped --network host \
  -v /opt/flashshell/openresty/conf:/etc/nginx/conf.d \
  -v /opt/flashshell/openresty/html:/usr/local/openresty/nginx/html \
  -v /opt/flashshell/openresty/certs:/etc/letsencrypt \
  openresty/openresty:alpine
docker ps --filter name=flashshell-openresty --format '{{.Names}} {{.Status}}'
`
	return s.execSSH(a.Server, script, 180*time.Second)
}

func (s *Service) handleWebProxy(_ context.Context, a WebCreateProxyArgs) (any, error) {
	conf := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    location / {
        proxy_pass http://%s;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
`, a.Domain, a.Upstream)
	remote := "/opt/flashshell/openresty/conf/" + sanitizeDomain(a.Domain) + ".conf"
	if err := s.writeRemoteText(a.Server, remote, conf); err != nil {
		return nil, err
	}
	reload := `docker exec flashshell-openresty openresty -t && docker exec flashshell-openresty openresty -s reload || nginx -t && nginx -s reload`
	res, err := s.execSSH(a.Server, reload, 30*time.Second)
	if err != nil {
		return nil, err
	}
	_ = s.ledger.UpsertSite(SiteRecord{
		Server: a.Server, Domain: a.Domain, Kind: "proxy", Upstream: a.Upstream,
		Enabled: true, CreatedAt: time.Now(),
	})
	return map[string]any{"ok": true, "domain": a.Domain, "upstream": a.Upstream, "reload": res}, nil
}

func (s *Service) handleWebStatic(_ context.Context, a WebCreateStaticArgs) (any, error) {
	root := "/opt/flashshell/openresty/html/" + sanitizeDomain(a.Domain)
	conf := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.html;
    location / { try_files $uri $uri/ /index.html; }
}
`, a.Domain, root)
	remote := "/opt/flashshell/openresty/conf/" + sanitizeDomain(a.Domain) + ".conf"
	mkdir := "mkdir -p " + shellQuote(root)
	if _, err := s.execSSH(a.Server, mkdir, 15*time.Second); err != nil {
		return nil, err
	}
	if err := s.writeRemoteText(a.Server, remote, conf); err != nil {
		return nil, err
	}
	reload := `docker exec flashshell-openresty openresty -t && docker exec flashshell-openresty openresty -s reload || true`
	res, err := s.execSSH(a.Server, reload, 30*time.Second)
	if err != nil {
		return nil, err
	}
	_ = s.ledger.UpsertSite(SiteRecord{
		Server: a.Server, Domain: a.Domain, Kind: "static", Root: root,
		Enabled: true, CreatedAt: time.Now(),
	})
	return map[string]any{"ok": true, "domain": a.Domain, "root": root, "reload": res}, nil
}

func (s *Service) handleWebSSL(_ context.Context, a WebIssueSslArgs) (any, error) {
	renew := "true"
	if a.AutoRenew != nil && !*a.AutoRenew {
		renew = "false"
	}
	script := fmt.Sprintf(`set -e
if ! command -v certbot >/dev/null; then
  if command -v apt-get >/dev/null; then apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y certbot; fi
fi
certbot certonly --webroot -w /opt/flashshell/openresty/html/%s -d %s --email %s --agree-tos --non-interactive
echo auto_renew=%s
`, sanitizeDomain(a.Domain), a.Domain, a.Email, renew)
	res, err := s.execSSH(a.Server, script, 180*time.Second)
	if err != nil {
		return nil, err
	}
	sites := s.ledger.ListSites(a.Server)
	for _, st := range sites {
		if st.Domain == a.Domain {
			st.Cert = true
			_ = s.ledger.UpsertSite(st)
		}
	}
	return map[string]any{"ok": true, "domain": a.Domain, "result": res}, nil
}

func sanitizeDomain(d string) string {
	d = strings.TrimSpace(strings.ToLower(d))
	d = strings.ReplaceAll(d, "/", "")
	d = strings.ReplaceAll(d, "..", "")
	return d
}

func (s *Service) writeRemoteText(server, path, content string) error {
	return s.handleWriteViaSftp(server, path, content)
}

func (s *Service) handleWriteViaSftp(server, path, content string) error {
	_, err := s.handleSftpWrite(context.Background(), SftpWriteArgs{
		Server:  server,
		Path:    path,
		Content: &content,
	})
	return err
}
