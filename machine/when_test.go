package machine

import "testing"

func TestEvaluateWhenEmpty(t *testing.T) {
	ok, err := EvaluateWhen("", nil)
	if err != nil || !ok {
		t.Fatalf("empty when should be true, got %v err=%v", ok, err)
	}
}

func TestEvaluateWhenVars(t *testing.T) {
	vars := map[string]string{"env": "prod", "flag": "1"}
	cases := []struct {
		expr string
		want bool
	}{
		{"${env} == prod", true},
		{"$env != prod", false},
		{"${env} == dev", false},
		{"${flag} == 1 && ${env} == prod", true},
		{"${flag} == 1 || ${env} == dev", true},
		{"!false", true},
		{"no", false},
	}
	for _, tc := range cases {
		ok, err := EvaluateWhen(tc.expr, vars)
		if err != nil {
			t.Fatalf("expr %q: %v", tc.expr, err)
		}
		if ok != tc.want {
			t.Fatalf("expr %q: want %v got %v", tc.expr, tc.want, ok)
		}
	}
}
