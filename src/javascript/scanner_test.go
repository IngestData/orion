package javascript

import (
	"strings"
	"testing"

	"github.com/Ingestdata/orion/src/scan"
)

func hasRule(findings []scan.Finding, rule string) bool {
	for _, f := range findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestRules_Hit(t *testing.T) {
	cases := []struct {
		rule string
		code string
	}{
		{"network-request", `fetch("https://evil.example.com");`},
		{"network-module", `const http = require("http");`},
		{"dynamic-exec", `eval("2 + 2");`},
		{"unbounded-loop", `while (true) { doWork(); }`},
		{"system-access", `const fs = require("fs");`},
		{"prototype-pollution", `record.__proto__ = {};`},
		{"hardcoded-secret", `const apiKey = "sk_live_abcdefghijklmnop";`},
	}
	for _, c := range cases {
		t.Run(c.rule, func(t *testing.T) {
			findings := New().Scan(c.code)
			if !hasRule(findings, c.rule) {
				t.Errorf("expected rule %q to fire on %q, got findings: %+v", c.rule, c.code, findings)
			}
		})
	}
}

func TestRules_Miss(t *testing.T) {
	code := `
function transform(record) {
  record.amountCents = Math.round(record.amount * 100);
  delete record.amount;
  return record;
}
`
	findings := New().Scan(code)
	if len(findings) != 0 {
		t.Errorf("expected no findings on benign code, got: %+v", findings)
	}
}

func TestCriticalRules_SkippedInComments(t *testing.T) {
	code := `
// eval("this is just an example in a docstring");
function transform(record) { return record; }
`
	findings := New().Scan(code)
	if hasRule(findings, "dynamic-exec") {
		t.Errorf("expected dynamic-exec to be skipped inside a comment, got: %+v", findings)
	}
}

func TestWarningRule_StillFiresInComments(t *testing.T) {
	code := `// password: "hardcoded1234567890"`
	findings := New().Scan(code)
	if !hasRule(findings, "hardcoded-secret") {
		t.Errorf("expected hardcoded-secret (a Warning) to still fire inside a comment, got: %+v", findings)
	}
}

func TestSizeLimit(t *testing.T) {
	code := strings.Repeat("a", maxTransformerLength+1)
	findings := New().Scan(code)
	if !hasRule(findings, "size-limit") {
		t.Errorf("expected size-limit finding on an oversized transformer, got: %+v", findings)
	}
}

func TestUnsanitizedSensitiveField(t *testing.T) {
	t.Run("flagged without sanitization", func(t *testing.T) {
		findings := New().Scan(`function transform(r) { return r.email; }`)
		if !hasRule(findings, "unsanitized-sensitive-field") {
			t.Errorf("expected unsanitized-sensitive-field finding, got: %+v", findings)
		}
	})

	t.Run("not flagged once masked", func(t *testing.T) {
		findings := New().Scan(`function transform(r) { r.email = mask(r.email); return r; }`)
		if hasRule(findings, "unsanitized-sensitive-field") {
			t.Errorf("did not expect unsanitized-sensitive-field once a mask() call is present, got: %+v", findings)
		}
	})
}

func TestPassed(t *testing.T) {
	if !scan.Passed(nil) {
		t.Error("expected Passed(nil) to be true")
	}
	if !scan.Passed([]scan.Finding{{Severity: scan.SeverityWarning}}) {
		t.Error("expected a Warning-only finding set to pass")
	}
	if scan.Passed([]scan.Finding{{Severity: scan.SeverityCritical}}) {
		t.Error("expected a Critical finding to fail Passed")
	}
}
