// Package r is Orion's R language module - a scan.Language implementation backed by the real
// lexer/parser in src/r/v1, rather than the line-oriented regex approach src/javascript uses.
// Working from an AST means the rules below never trip over R code sitting inside a comment
// (the lexer discards comments before the parser ever sees them, so there's no separate
// comment-skipping logic to get wrong), and can ask precise structural questions a regex can't,
// like "does this `while (TRUE)` loop contain a `break` anywhere in its own body" rather than
// just "does the literal text `while (true)` appear".
package r

import (
	"fmt"
	"regexp"

	v1 "github.com/Ingestdata/orion/src/r/v1"
	"github.com/Ingestdata/orion/src/scan"
)

type Scanner struct{}

func New() Scanner {
	return Scanner{}
}

const maxTransformerLength = 20_000

// callRule maps a resolved call target (see callName) to the finding it should raise.
type callRule struct {
	name     string
	severity scan.Severity
	message  string
}

var networkRule = callRule{"network-call", scan.SeverityCritical, "Transformer code may not make network requests."}
var systemRule = callRule{"system-access", scan.SeverityCritical, "Transformer code may not spawn processes, touch the filesystem, or read/write the host environment."}
var evalRule = callRule{"dynamic-exec", scan.SeverityCritical, "Transformer code may not dynamically evaluate constructed code (eval)."}

var callRules = map[string]callRule{
	"url":                     networkRule,
	"download.file":           networkRule,
	"socketConnection":        networkRule,
	"curl::curl":              networkRule,
	"curl::curl_fetch_memory": networkRule,
	"httr::GET":               networkRule,
	"httr::POST":              networkRule,
	"httr::PUT":               networkRule,
	"httr::DELETE":            networkRule,
	"httr::VERB":              networkRule,
	"RCurl::getURL":           networkRule,
	"RCurl::getURLContent":    networkRule,

	"system":       systemRule,
	"system2":      systemRule,
	"shell":        systemRule,
	"shell.exec":   systemRule,
	"unlink":       systemRule,
	"file.remove":  systemRule,
	"Sys.setenv":   systemRule,
	"Sys.unsetenv": systemRule,

	"eval": evalRule,
}

var secretNamePattern = regexp.MustCompile(`(?i)^(api[_.]?key|secret|token|password|passwd|credential)s?$`)
var sensitiveFieldPattern = regexp.MustCompile(`(?i)^(email|ssn|social[_.]?security|phone|credit[_.]?card|password)$`)
var sanitizerCallPattern = regexp.MustCompile(`(?i)^(gsub|sub|mask\w*|redact\w*|hash\w*|digest(::digest)?)$`)

const minSecretLiteralLen = 16

// Scan lexes, parses, and walks the transformer source, reporting every rule violation found.
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

	prog, errs := v1.Parse(code)
	if len(errs) > 0 {
		// The parser is a young prototype, so a syntax error is advisory rather than a hard
		// block: it may just as well be a gap in Orion's own R grammar coverage as a problem
		// with the transformer. Whatever parsed before the first error is still walked below.
		findings = append(findings, scan.Finding{
			Rule:     "parse-incomplete",
			Severity: scan.SeverityWarning,
			Message:  fmt.Sprintf("R source could not be fully parsed (%s); analysis below may be incomplete.", errs[0]),
			Line:     0,
		})
	}

	w := &walker{}
	for _, stmt := range prog.Body {
		w.walk(stmt, &findings)
	}

	if w.sawSensitiveField && !w.sawSanitizerCall {
		findings = append(findings, scan.Finding{
			Rule:     "unsanitized-sensitive-field",
			Severity: scan.SeverityWarning,
			Message:  "Code references a sensitive-looking field (email/ssn/phone/credit card/password) but no masking, redaction, or hashing call was found anywhere in the transformer.",
			Line:     0,
		})
	}

	return findings
}

type walker struct {
	sawSensitiveField bool
	sawSanitizerCall  bool
}

// walk visits n and its descendants, appending findings as rules match. Loop bodies run their
// own bounded containsBreak search (below) rather than relying on this traversal, so that a
// break found deep inside an unrelated nested loop never gets credited to an outer one.
func (w *walker) walk(n v1.Node, findings *[]scan.Finding) {
	if n == nil {
		return
	}

	switch node := n.(type) {
	case *v1.Ident:
		if sensitiveFieldPattern.MatchString(node.Name) {
			w.sawSensitiveField = true
		}

	case *v1.CallExpr:
		if name, ok := callName(node.Fun); ok {
			if sanitizerCallPattern.MatchString(name) {
				w.sawSanitizerCall = true
			}
			if rule, ok := callRules[name]; ok {
				*findings = append(*findings, scan.Finding{
					Rule:     rule.name,
					Severity: rule.severity,
					Message:  rule.message,
					Line:     0,
				})
			}
		}
		w.walk(node.Fun, findings)
		for _, a := range node.Args {
			if secretNamePattern.MatchString(a.Name) {
				if lit, ok := a.Value.(*v1.StringLit); ok && len(lit.Value) >= minSecretLiteralLen {
					*findings = append(*findings, hardcodedSecretFinding())
				}
			}
			w.walk(a.Value, findings)
		}

	case *v1.AssignExpr:
		if lhs, ok := node.Lhs.(*v1.Ident); ok && secretNamePattern.MatchString(lhs.Name) {
			if lit, ok := node.Rhs.(*v1.StringLit); ok && len(lit.Value) >= minSecretLiteralLen {
				*findings = append(*findings, hardcodedSecretFinding())
			}
		}
		w.walk(node.Lhs, findings)
		w.walk(node.Rhs, findings)

	case *v1.WhileExpr:
		if isAlwaysTrue(node.Cond) && !containsBreak(node.Body) {
			*findings = append(*findings, scan.Finding{
				Rule:     "unbounded-loop",
				Severity: scan.SeverityCritical,
				Message:  "while (TRUE) loop with no reachable break could hang the transformer runtime.",
				Line:     0,
			})
		}
		w.walk(node.Cond, findings)
		w.walk(node.Body, findings)

	case *v1.RepeatExpr:
		if !containsBreak(node.Body) {
			*findings = append(*findings, scan.Finding{
				Rule:     "unbounded-loop",
				Severity: scan.SeverityCritical,
				Message:  "repeat loop with no reachable break could hang the transformer runtime.",
				Line:     0,
			})
		}
		w.walk(node.Body, findings)

	case *v1.UnaryExpr:
		w.walk(node.X, findings)
	case *v1.BinaryExpr:
		w.walk(node.X, findings)
		w.walk(node.Y, findings)
	case *v1.IndexExpr:
		w.walk(node.X, findings)
		for _, a := range node.Args {
			w.walk(a.Value, findings)
		}
	case *v1.DollarExpr:
		if sensitiveFieldPattern.MatchString(node.Name) {
			w.sawSensitiveField = true
		}
		w.walk(node.X, findings)
	case *v1.AtExpr:
		if sensitiveFieldPattern.MatchString(node.Name) {
			w.sawSensitiveField = true
		}
		w.walk(node.X, findings)
	case *v1.FunctionDef:
		for _, p := range node.Params {
			w.walk(p.Default, findings)
		}
		w.walk(node.Body, findings)
	case *v1.IfExpr:
		w.walk(node.Cond, findings)
		w.walk(node.Then, findings)
		w.walk(node.Else, findings)
	case *v1.ForExpr:
		w.walk(node.Seq, findings)
		w.walk(node.Body, findings)
	case *v1.BlockExpr:
		for _, s := range node.Stmts {
			w.walk(s, findings)
		}
	}
}

func hardcodedSecretFinding() scan.Finding {
	return scan.Finding{
		Rule:     "hardcoded-secret",
		Severity: scan.SeverityWarning,
		Message:  "Possible hardcoded credential literal - use a project secret instead.",
		Line:     0,
	}
}

// callName resolves a call's callee expression to the dotted name used to key callRules:
// a bare identifier ("url"), or a namespaced reference ("httr::GET"). Any other callee shape
// (a $ / @ access, or the result of another call) isn't resolved to a static name and is
// reported as not-ok, since R's dynamism makes it unsafe to assume otherwise.
func callName(fun v1.Node) (string, bool) {
	switch f := fun.(type) {
	case *v1.Ident:
		return f.Name, true
	case *v1.NamespaceExpr:
		return f.Pkg + "::" + f.Name, true
	default:
		return "", false
	}
}

// isAlwaysTrue reports whether cond is the literal TRUE or the literal 1, the two idiomatic
// spellings of an intentionally-unbounded while loop in R.
func isAlwaysTrue(cond v1.Node) bool {
	switch c := cond.(type) {
	case *v1.ConstLit:
		return c.Kind == v1.TRUE
	case *v1.NumberLit:
		return c.Value == "1"
	default:
		return false
	}
}

// containsBreak reports whether a break statement reachable by this loop appears anywhere in
// body. It does not descend into nested FunctionDef bodies: a break written inside a closure
// defined within the loop can't actually unwind the enclosing loop in R, so crediting it would
// produce a false negative on a genuinely unbounded loop.
func containsBreak(body v1.Node) bool {
	if body == nil {
		return false
	}
	switch n := body.(type) {
	case *v1.BreakStmt:
		return true
	case *v1.BlockExpr:
		for _, s := range n.Stmts {
			if containsBreak(s) {
				return true
			}
		}
		return false
	case *v1.IfExpr:
		return containsBreak(n.Then) || containsBreak(n.Else)
	case *v1.ForExpr:
		return containsBreak(n.Body)
	case *v1.WhileExpr:
		return containsBreak(n.Body)
	case *v1.RepeatExpr:
		return containsBreak(n.Body)
	case *v1.FunctionDef:
		return false // a nested function's own break can't reach this loop
	default:
		return false
	}
}
