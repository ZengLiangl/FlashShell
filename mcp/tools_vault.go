package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) handleSaveCredential(_ context.Context, a SaveCredentialArgs) (any, error) {
	if _, err := s.machineByAlias(a.Server); err != nil {
		return nil, err
	}
	pub := map[string]string{}
	sec := map[string]string{}
	for k, v := range a.Fields {
		if isSecretField(k, a.SecretFields) {
			sec[k] = v
		} else {
			pub[k] = v
		}
	}
	for k, vid := range a.FieldsFromVault {
		val, ok := s.vault.SecretValue(vid)
		if !ok {
			return nil, wrapErr("[notfound]", "fieldsFromVault 找不到 "+vid)
		}
		sec[k] = val
	}
		item := VaultItem{
		ServerAlias: a.Server,
		Kind:        a.Kind,
		Label:       a.Label,
		Public:      pub,
		Secrets:     sec,
	}
	if item.Public == nil {
		item.Public = map[string]string{}
	}
	item.Public["__tunnel_server_id"] = a.Server
	if a.Notes != nil {
		item.Notes = *a.Notes
	}
	saved, err := s.vault.Save(item)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": saved.ID, "label": saved.Label, "kind": saved.Kind, "serverAlias": saved.ServerAlias}, nil
}

func isSecretField(k string, listed []string) bool {
	lk := strings.ToLower(k)
	if strings.Contains(lk, "password") || strings.Contains(lk, "secret") || strings.Contains(lk, "token") || lk == "pass" {
		return true
	}
	for _, s := range listed {
		if strings.EqualFold(s, k) {
			return true
		}
	}
	return false
}

func (s *Service) handleInstallWithSecret(ctx context.Context, a InstallWithSecretArgs) (any, error) {
	if hit, why := commandBlocked(a.InstallScript); hit {
		return nil, wrapErr("[blocked]", why)
	}
	values := map[string]string{}
	for k, v := range a.Public {
		values[k] = v
	}
	secretsPlain := map[string]string{}
	for k, spec := range a.Secrets {
		val, err := generateSecret(spec)
		if err != nil {
			return nil, err
		}
		secretsPlain[k] = val
		values[k] = val
	}
	script := replacePlaceholders(a.InstallScript, values)
	to := clampTimeout(a.TimeoutSecs, 600, 1, 1800)
	res, err := s.execSSH(a.Server, script, to)
	if err != nil {
		return nil, err
	}
	if a.VerifyScript != nil && strings.TrimSpace(*a.VerifyScript) != "" {
		vs := replacePlaceholders(*a.VerifyScript, values)
		vr, err := s.execSSH(a.Server, vs, to)
		if err != nil {
			return nil, err
		}
		if vr.ExitCode != 0 {
			return nil, fmt.Errorf("verifyScript 失败 exit=%d stderr=%s", vr.ExitCode, clip(vr.Stderr, 400))
		}
	}
	secSave := map[string]string{}
	pubSave := map[string]string{}
	for _, f := range a.SaveFields {
		if v, ok := secretsPlain[f]; ok {
			secSave[f] = v
			continue
		}
		if v, ok := values[f]; ok {
			if isSecretField(f, nil) {
				secSave[f] = v
			} else {
				pubSave[f] = v
			}
		}
	}
	item := VaultItem{
		ServerAlias: a.Server,
		Kind:        a.Kind,
		Label:       a.Label,
		Public:      pubSave,
		Secrets:     secSave,
	}
	if a.Notes != nil {
		item.Notes = *a.Notes
	}
	if a.InstallPath != nil {
		item.InstallPath = *a.InstallPath
	}
	saved, err := s.vault.Save(item)
	if err != nil {
		return nil, err
	}
	stdout := clip(res.Stdout, 500)
	return map[string]any{
		"vault_id": saved.ID,
		"label":    saved.Label,
		"kind":     saved.Kind,
		"public":   pubSave,
		"stdout":   stdout,
		"exitCode": res.ExitCode,
	}, nil
}

func replacePlaceholders(script string, values map[string]string) string {
	out := script
	for k, v := range values {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func generateSecret(spec SecretSpec) (string, error) {
	switch spec.Kind {
	case "random_hex":
		n := spec.Length
		if n <= 0 {
			n = 16
		}
		b := make([]byte, n)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		return hex.EncodeToString(b)[:n], nil
	case "uuid":
		return uuid.NewString(), nil
	case "random_port":
		min, max := spec.Min, spec.Max
		if min <= 0 {
			min = 20000
		}
		if max <= min {
			max = 40000
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d", min+int(n.Int64())), nil
	default: // random_string
		n := spec.Length
		if n <= 0 {
			n = 24
		}
		alphabet := spec.Alphabet
		chars := "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
		if alphabet == "strong" {
			chars = chars + "!@#$%^&*()-_=+"
		}
		b := make([]byte, n)
		for i := 0; i < n; i++ {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			if err != nil {
				return "", err
			}
			b[i] = chars[idx.Int64()]
		}
		return string(b), nil
	}
}

func (s *Service) handleInstallApp(ctx context.Context, a InstallAppArgs) (any, error) {
	app := strings.ToLower(strings.TrimSpace(a.App))
	port := 0
	if a.Port != nil {
		port = *a.Port
	}
	version := "latest"
	if a.Version != nil && *a.Version != "" {
		version = *a.Version
	}
	script, kind, label, defPort, err := installAppScript(app, port, version)
	if err != nil {
		return nil, err
	}
	spec := SecretSpec{Kind: "random_string", Length: 24, Alphabet: "strong"}
	pass, err := generateSecret(spec)
	if err != nil {
		return nil, err
	}
	script = strings.ReplaceAll(script, "__PASSWORD__", pass)
	res, err := s.execSSH(a.Server, script, clampTimeout(nil, 600, 1, 1800))
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("安装失败 exit=%d stderr=%s", res.ExitCode, clip(res.Stderr, 500))
	}
	item := VaultItem{
		ServerAlias: a.Server,
		Kind:        kind,
		Label:       label,
		Public: map[string]string{
			"host":                 "127.0.0.1",
			"port":                 fmt.Sprintf("%d", defPort),
			"user":                 defaultDBUser(app),
			"__tunnel_server_id":   a.Server,
		},
		Secrets: map[string]string{"password": pass},
	}
	if kind == "redis_conn" {
		item.Public["db"] = "0"
		delete(item.Public, "user")
	}
	saved, err := s.vault.Save(item)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"vault_id": saved.ID,
		"label":    saved.Label,
		"kind":     saved.Kind,
		"port":     defPort,
		"stdout":   clip(res.Stdout, 500),
	}, nil
}

func defaultDBUser(app string) string {
	if app == "postgres" {
		return "postgres"
	}
	if app == "mongodb" {
		return "root"
	}
	return "root"
}

func installAppScript(app string, port int, version string) (script, kind, label string, defPort int, err error) {
	name := "flashshell-" + app
	switch app {
	case "mysql":
		if port == 0 {
			port = 3306
		}
		kind, label, defPort = "mysql_conn", "MySQL", port
		script = fmt.Sprintf(`docker rm -f %s 2>/dev/null || true
docker run -d --name %s --restart unless-stopped -p 127.0.0.1:%d:3306 -e MYSQL_ROOT_PASSWORD='__PASSWORD__' mysql:%s --character-set-server=utf8mb4
`, name, name, port, version)
	case "redis":
		if port == 0 {
			port = 6379
		}
		kind, label, defPort = "redis_conn", "Redis", port
		script = fmt.Sprintf(`docker rm -f %s 2>/dev/null || true
docker run -d --name %s --restart unless-stopped -p 127.0.0.1:%d:6379 redis:%s redis-server --requirepass '__PASSWORD__'
`, name, name, port, version)
	case "postgres":
		if port == 0 {
			port = 5432
		}
		kind, label, defPort = "postgres_conn", "PostgreSQL", port
		script = fmt.Sprintf(`docker rm -f %s 2>/dev/null || true
docker run -d --name %s --restart unless-stopped -p 127.0.0.1:%d:5432 -e POSTGRES_PASSWORD='__PASSWORD__' postgres:%s
`, name, name, port, version)
	case "mongodb":
		if port == 0 {
			port = 27017
		}
		kind, label, defPort = "mongodb_conn", "MongoDB", port
		script = fmt.Sprintf(`docker rm -f %s 2>/dev/null || true
docker run -d --name %s --restart unless-stopped -p 127.0.0.1:%d:27017 -e MONGO_INITDB_ROOT_USERNAME=root -e MONGO_INITDB_ROOT_PASSWORD='__PASSWORD__' mongo:%s
`, name, name, port, version)
	case "openresty":
		if port == 0 {
			port = 80
		}
		kind, label, defPort = "openresty", "OpenResty", port
		script = fmt.Sprintf(`mkdir -p /opt/flashshell/openresty/conf /opt/flashshell/openresty/html /opt/flashshell/openresty/certs
docker rm -f flashshell-openresty 2>/dev/null || true
docker run -d --name flashshell-openresty --restart unless-stopped --network host -v /opt/flashshell/openresty/conf:/etc/nginx/conf.d -v /opt/flashshell/openresty/html:/usr/local/openresty/nginx/html -v /opt/flashshell/openresty/certs:/etc/letsencrypt openresty/openresty:%s
`, version)
	default:
		err = wrapErr("[notfound]", "未知应用 "+app+"，支持 mysql/redis/postgres/mongodb/openresty")
	}
	return
}
