package mcp

import (
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type SiteRecord struct {
	Server    string    `yaml:"server" json:"server"`
	Domain    string    `yaml:"domain" json:"domain"`
	Kind      string    `yaml:"kind" json:"kind"` // proxy | static
	Upstream  string    `yaml:"upstream,omitempty" json:"upstream,omitempty"`
	Root      string    `yaml:"root,omitempty" json:"root,omitempty"`
	Enabled   bool      `yaml:"enabled" json:"enabled"`
	Cert      bool      `yaml:"cert" json:"cert"`
	CreatedAt time.Time `yaml:"createdAt" json:"createdAt"`
}

type DeployTarget struct {
	Name               string            `yaml:"name" json:"name"`
	Recipe             string            `yaml:"recipe" json:"recipe"`
	Servers            []string          `yaml:"servers,omitempty" json:"servers,omitempty"`
	Domain             string            `yaml:"domain,omitempty" json:"domain,omitempty"`
	HTTPS              bool              `yaml:"https,omitempty" json:"https,omitempty"`
	Workdir            string            `yaml:"workdir,omitempty" json:"workdir,omitempty"`
	BuildSource        string            `yaml:"buildSource,omitempty" json:"buildSource,omitempty"`
	BuildCommands      []string          `yaml:"buildCommands,omitempty" json:"buildCommands,omitempty"`
	AutoRollback       bool              `yaml:"autoRollback,omitempty" json:"autoRollback,omitempty"`
	SkipUnchangedBuild *bool             `yaml:"skipUnchangedBuild,omitempty" json:"skipUnchangedBuild,omitempty"`
	Vars               map[string]string `yaml:"vars,omitempty" json:"vars,omitempty"`
	Artifact           *DtArtifactArg    `yaml:"artifact,omitempty" json:"artifact,omitempty"`
	Compose            *DtComposeArg     `yaml:"compose,omitempty" json:"compose,omitempty"`
	Health             *DtHealthArg      `yaml:"health,omitempty" json:"health,omitempty"`
	Image              *DtImageArg       `yaml:"image,omitempty" json:"image,omitempty"`
	Release            *DtReleaseArg     `yaml:"release,omitempty" json:"release,omitempty"`
	UpdatedAt          time.Time         `yaml:"updatedAt" json:"updatedAt"`
}

type DeployHistoryItem struct {
	Target    string    `yaml:"target" json:"target"`
	Version   string    `yaml:"version" json:"version"`
	Servers   []string  `yaml:"servers" json:"servers"`
	OK        bool      `yaml:"ok" json:"ok"`
	Running   bool      `yaml:"running" json:"running"`
	Note      string    `yaml:"note,omitempty" json:"note,omitempty"`
	Detail    string    `yaml:"detail,omitempty" json:"detail,omitempty"`
	ImageSize string    `yaml:"imageSize,omitempty" json:"imageSize,omitempty"`
	Time      time.Time `yaml:"time" json:"time"`
}

type Ledger struct {
	mu      sync.Mutex
	sites   []SiteRecord
	targets []DeployTarget
	history []DeployHistoryItem
}

func loadLedger() *Ledger {
	l := &Ledger{}
	l.sites = readYAMLSlice[SiteRecord](sitesFile)
	l.targets = readYAMLSlice[DeployTarget](deploysFile)
	l.history = readYAMLSlice[DeployHistoryItem](historyFile)
	return l
}

func readYAMLSlice[T any](name string) []T {
	root, err := homeDir()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(join(root, name))
	if err != nil {
		return nil
	}
	var v []T
	if yaml.Unmarshal(b, &v) != nil {
		return nil
	}
	return v
}

func writeYAMLSlice[T any](name string, v []T) error {
	root, err := homeDir()
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(join(root, name), b, 0600)
}

func (l *Ledger) ListSites(server string) []SiteRecord {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []SiteRecord
	for _, s := range l.sites {
		if server == "" || s.Server == server {
			out = append(out, s)
		}
	}
	if out == nil {
		out = []SiteRecord{}
	}
	return out
}

func (l *Ledger) UpsertSite(s SiteRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, old := range l.sites {
		if old.Server == s.Server && old.Domain == s.Domain {
			l.sites[i] = s
			return writeYAMLSlice(sitesFile, l.sites)
		}
	}
	l.sites = append(l.sites, s)
	return writeYAMLSlice(sitesFile, l.sites)
}

func (l *Ledger) UpsertTarget(t DeployTarget) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	t.UpdatedAt = time.Now()
	for i, old := range l.targets {
		if old.Name == t.Name {
			l.targets[i] = t
			return writeYAMLSlice(deploysFile, l.targets)
		}
	}
	l.targets = append(l.targets, t)
	return writeYAMLSlice(deploysFile, l.targets)
}

func (l *Ledger) GetTarget(name string) (DeployTarget, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, t := range l.targets {
		if t.Name == name {
			return t, true
		}
	}
	return DeployTarget{}, false
}

func (l *Ledger) AddHistory(h DeployHistoryItem) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if h.Time.IsZero() {
		h.Time = time.Now()
	}
	l.history = append(l.history, h)
	return writeYAMLSlice(historyFile, l.history)
}

func (l *Ledger) History(target string) []DeployHistoryItem {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []DeployHistoryItem
	for i := len(l.history) - 1; i >= 0; i-- {
		if l.history[i].Target == target {
			out = append(out, l.history[i])
		}
	}
	if out == nil {
		out = []DeployHistoryItem{}
	}
	return out
}
