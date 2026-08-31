package mcp

import "testing"

func TestCommandBlocked(t *testing.T) {
	if !commandBlocked("rm -rf /") {
		t.Fatal("rm -rf / should be blocked")
	}
	if !commandBlocked("mkfs.ext4 /dev/sda") {
		t.Fatal("mkfs should be blocked")
	}
	if commandBlocked("ls -la /var/log") {
		t.Fatal("ls should be allowed")
	}
}

func TestPathBlocked(t *testing.T) {
	if !pathBlocked("/etc/shadow") {
		t.Fatal("shadow should be blocked")
	}
	if !pathBlocked("/home/u/.ssh/id_rsa") {
		t.Fatal(".ssh should be blocked")
	}
	if pathBlocked("/opt/app/config.yaml") {
		t.Fatal("yaml config should be allowed")
	}
}

func TestBuiltinSkillCount(t *testing.T) {
	want := []string{
		"1panel-ops", "borgbackup-restic", "caddy-traefik", "certbot-acme", "clickhouse-ops",
		"docker-ps", "git-server", "iptables-firewalld", "java-jvm-tuning",
		"jenkins-runner", "kafka-ops", "kubernetes-basics", "linux-fundamentals", "log-investigation",
		"loki-promtail", "minio-s3", "mongodb-ops", "mysql-ops", "nfs-smb-share",
		"nginx-status", "nodejs-pm2", "ollama-vllm", "openresty-lua", "php-fpm-ops",
		"portainer-ops", "postgresql-ops", "prometheus-grafana", "python-venv-uwsgi", "rabbitmq-ops",
		"redis-ops", "reeve-app-deploy", "ruoyi-plus-uniapp-deploy", "selinux-apparmor",
		"ssh-hardening", "systemd-service", "wireguard-vpn",
	}
	got := map[string]bool{}
	for _, s := range builtinSkills() {
		got[s.Name] = true
	}
	if len(builtinSkills()) != len(want) {
		t.Fatalf("skill count %d want %d", len(builtinSkills()), len(want))
	}
	for _, n := range want {
		if !got[n] {
			t.Fatalf("missing skill %s", n)
		}
	}
}
