package mcp

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

type SkillHit struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Description string   `json:"description"`
	Triggers    []string `json:"triggers"`
	HitCount    int      `json:"hitCount"`
}

type Knowledge struct{}

func newKnowledge() *Knowledge {
	_ = mustSubdir(userSkillsDir)
	_ = mustSubdir(experienceDir)
	_ = mustSubdir(runbooksDir)
	return &Knowledge{}
}

func skillNameOK(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func (k *Knowledge) ListSkills() []map[string]any {
	var out []map[string]any
	for _, s := range builtinSkills() {
		out = append(out, map[string]any{
			"name":        s.Name,
			"source":      "builtin",
			"description": s.Description,
			"triggers":    s.Triggers,
		})
	}
	dir := mustSubdir(userSkillsDir)
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		body, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		meta := parseSkillMeta(string(body))
		out = append(out, map[string]any{
			"name":        name,
			"source":      "user",
			"description": meta.Description,
			"triggers":    meta.Triggers,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out
}

func (k *Knowledge) ListSkillNames() []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range builtinSkills() {
		if _, ok := seen[s.Name]; ok {
			continue
		}
		seen[s.Name] = struct{}{}
		out = append(out, s.Name)
	}
	dir := mustSubdir(userSkillsDir)
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if out == nil {
		out = []string{}
	}
	sort.Strings(out)
	return out
}

func (k *Knowledge) GetSkill(name string) (string, error) {
	if !skillNameOK(name) {
		return "", blockedErr("[notfound]", "技能名非法")
	}
	user := filepath.Join(mustSubdir(userSkillsDir), name+".md")
	if b, err := os.ReadFile(user); err == nil {
		return string(b), nil
	}
	for _, s := range builtinSkills() {
		if s.Name == name {
			return s.Body, nil
		}
	}
	return "", blockedErr("[notfound]", "未找到技能 "+name)
}

func (k *Knowledge) Evaluate(prompt string) []SkillHit {
	p := strings.ToLower(prompt)
	var hits []SkillHit
	add := func(name, source, desc string, triggers []string) {
		n := 0
		for _, t := range triggers {
			if t != "" && strings.Contains(p, strings.ToLower(t)) {
				n++
			}
		}
		if n == 0 {
			return
		}
		hits = append(hits, SkillHit{Name: name, Source: source, Description: desc, Triggers: triggers, HitCount: n})
	}
	for _, s := range builtinSkills() {
		add(s.Name, "builtin", s.Description, s.Triggers)
	}
	dir := mustSubdir(userSkillsDir)
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		body, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		meta := parseSkillMeta(string(body))
		add(name, "user", meta.Description, meta.Triggers)
	}
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			pi, pj := sourcePri(hits[i].Source), sourcePri(hits[j].Source)
			if hits[j].HitCount > hits[i].HitCount || (hits[j].HitCount == hits[i].HitCount && pj < pi) {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
	return hits
}

func sourcePri(s string) int {
	switch s {
	case "project":
		return 0
	case "user":
		return 1
	default:
		return 2
	}
}

func (k *Knowledge) Recall(query string) []string {
	dir := mustSubdir(experienceDir)
	ents, _ := os.ReadDir(dir)
	q := strings.ToLower(strings.TrimSpace(query))
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		text := string(b)
		if q == "" {
			line := firstLine(text)
			out = append(out, e.Name()+": "+line)
			continue
		}
		if strings.Contains(strings.ToLower(text), q) {
			out = append(out, clip(text, 800))
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func (k *Knowledge) ListRunbooks() []string {
	dir := mustSubdir(runbooksDir)
	ents, _ := os.ReadDir(dir)
	var out []string
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), ".md"))
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func (k *Knowledge) GetRunbook(name string) (string, error) {
	if !skillNameOK(name) {
		return "", blockedErr("[notfound]", "runbook 名非法")
	}
	b, err := os.ReadFile(filepath.Join(mustSubdir(runbooksDir), name+".md"))
	if err != nil {
		return "", blockedErr("[notfound]", "未找到 runbook "+name)
	}
	return string(b), nil
}

func (k *Knowledge) AppendExperience(title, body string) error {
	dir := mustSubdir(experienceDir)
	name := strings.TrimSpace(title)
	if name == "" {
		name = uuid.NewString()[:8]
	}
	name = regexp.MustCompile(`[^A-Za-z0-9._-]+`).ReplaceAllString(name, "_")
	return os.WriteFile(filepath.Join(dir, name+".md"), []byte(body), 0644)
}

type skillMeta struct {
	Description string
	Triggers    []string
}

func parseSkillMeta(body string) skillMeta {
	m := skillMeta{}
	if strings.HasPrefix(body, "---") {
		if i := strings.Index(body[3:], "---"); i >= 0 {
			fm := body[3 : 3+i]
			for _, line := range strings.Split(fm, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "description:") {
					m.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
					m.Description = strings.Trim(m.Description, `"'`)
				}
				if strings.HasPrefix(line, "triggers:") {
					raw := strings.TrimSpace(strings.TrimPrefix(line, "triggers:"))
					raw = strings.Trim(raw, "[]")
					for _, p := range strings.Split(raw, ",") {
						p = strings.TrimSpace(p)
						p = strings.Trim(p, `"'`)
						if p != "" {
							m.Triggers = append(m.Triggers, p)
						}
					}
				}
			}
		}
	}
	if m.Description == "" {
		m.Description = firstLine(body)
	}
	return m
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

type builtinSkill struct {
	Name        string
	Description string
	Triggers    []string
	Body        string
}
