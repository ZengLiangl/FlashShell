package define

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStepListUnmarshal(t *testing.T) {
	raw := `
steps:
  - mvn clean
  - cmd: mvn test
    on_fail: continue
  - cmd: deploy
    retry: 2
`
	var cmd struct {
		Steps StepList `yaml:"steps"`
	}
	if err := yaml.Unmarshal([]byte(raw), &cmd); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(cmd.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(cmd.Steps))
	}
	if cmd.Steps[1].NormalizedOnFail() != OnFailContinue {
		t.Fatalf("expected continue on fail")
	}
	if cmd.Steps[2].MaxAttempts() != 3 {
		t.Fatalf("expected 3 attempts")
	}
}
