// Package javascript is Orion's first working language module - a static-analysis pass over
// JavaScript transformer source text. It is deliberately a line-oriented regex scanner, not a
// real JS parser/AST/data-flow analyzer: it catches the obvious, common-case attempts (a
// transformer calling out to the network, eval'ing a string, spinning forever) without the cost
// of embedding a full JS toolchain. False negatives are expected for obfuscated code; false
// positives are expected for the two heuristic (Warning-severity) rules below.
package javascript

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/Ingestdata/orion/src/scan"
)

type rule struct {
	name     string
	pattern  *regexp.Regexp
	severity scan.Severity
	message  string
}

var lineRules = []rule{
	{
		name:     "network-request",
		pattern:  regexp.MustCompile(`\b(fetch|XMLHttpRequest|WebSocket|axios(\.\w+)?)\s*\(`),
		severity: scan.SeverityCritical,
		message:  "Transformer code may not make network requests.",
	},
	{
		name:     "network-module",
		pattern:  regexp.MustCompile(`require\(\s*['"](http|https|net|dgram|dns)['"]\s*\)`),
		severity: scan.SeverityCritical,
		message:  "Transformer code may not import networking modules.",
	},
	{
		name:     "dynamic-exec",
		pattern:  regexp.MustCompile(`\beval\s*\(|\bnew\s+Function\s*\(`),
		severity: scan.SeverityCritical,
		message:  "Transformer code may not dynamically execute strings as code (eval / new Function).",
	},
	{
		name:     "unbounded-loop",
		pattern:  regexp.MustCompile(`while\s*\(\s*true\s*\)|for\s*\(\s*;\s*;\s*\)`),
		severity: scan.SeverityCritical,
		message:  "Unbounded loop construct could hang the transformer runtime.",
	},
	{
		name:     "system-access",
		pattern:  regexp.MustCompile(`require\(\s*['"](fs|child_process|os|cluster)['"]\s*\)|process\.(exit|binding|env)\b`),
		severity: scan.SeverityCritical,
		message:  "Transformer code may not access the filesystem, spawn processes, or read the host environment.",
	},
	{
		name:     "prototype-pollution",
		pattern:  regexp.MustCompile(`__proto__|constructor\s*\[\s*['"]prototype['"]\s*\]`),
		severity: scan.SeverityCritical,
		message:  "Direct prototype manipulation is not allowed.",
	},
	{
		name:     "hardcoded-secret",
		pattern:  regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password)\s*[:=]\s*['"][A-Za-z0-9_\-]{16,}['"]`),
		severity: scan.SeverityWarning,
		message:  "Possible hardcoded credential literal - use a project secret instead.",
	},
}

const maxTransformerLength = 20_000

// sensitiveFieldPattern flags common PII field names; sanitizationPattern looks for anything
// that suggests the code actually does something about them before they leave the transformer.
var sensitiveFieldPattern = regexp.MustCompile(`(?i)\b(email|ssn|social_security|phone|credit_?card|password)\b`)
var sanitizationPattern = regexp.MustCompile(`(?i)\.replace\s*\(|\bmask\w*\s*\(|\bredact\w*\s*\(|\bhash\w*\s*\(`)

type Scanner struct{}

func New() Scanner {
	return Scanner{}
}

// Scan runs every rule over the transformer source and returns its findings plus a size guard.
func (Scanner) Scan(code string) []scan.Finding {
	findings := []scan.Finding{}

	if len(code) > maxTransformerLength {
		findings = append(findings, scan.Finding{
			Rule:     "size-limit",
			Severity: scan.SeverityWarning,
			Message:  "Transformer is unusually large - consider splitting logic or simplifying it.",
			Line:     0,
		})
	}

	scanner := bufio.NewScanner(strings.NewReader(code))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// Skip line/block comment lines for the hard-blocking rules only, so a commented-out
		// example in a docstring doesn't itself get flagged - still scan comments for the
		// advisory heuristics below since a real secret left in a comment is still a leak.
		trimmed := strings.TrimSpace(line)
		isComment := strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*")

		for _, r := range lineRules {
			if isComment && r.severity == scan.SeverityCritical {
				continue
			}
			if r.pattern.MatchString(line) {
				findings = append(findings, scan.Finding{
					Rule:     r.name,
					Severity: r.severity,
					Message:  r.message,
					Line:     lineNo,
				})
			}
		}
	}

	if sensitiveFieldPattern.MatchString(code) && !sanitizationPattern.MatchString(code) {
		findings = append(findings, scan.Finding{
			Rule:     "unsanitized-sensitive-field",
			Severity: scan.SeverityWarning,
			Message:  "Code references a sensitive-looking field (email/ssn/phone/password) but no masking, redaction, or hashing call was found anywhere in the transformer.",
			Line:     0,
		})
	}

	return findings
}
