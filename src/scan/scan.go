// Package scan defines the shared contract every transformer-language scanner implements.
// Orion registers one Language per supported transformer language (see src/javascript for the
// reference implementation) in a plain map in main() - implement this interface and add an
// entry to that map to support a new language.
package scan

// Severity gates deployment: Critical findings block it, Warning findings are advisory only.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
)

type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Line     int      `json:"line"`
}

// Language scans a single transformer's source code and reports what it found. Implementations
// should be safe for concurrent use - Scan is called once per HTTP request, potentially
// concurrently, against a single shared instance.
type Language interface {
	Scan(code string) []Finding
}

// Passed reports whether a scan permits deployment: any Critical finding blocks it, Warning
// findings are surfaced but non-blocking.
func Passed(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			return false
		}
	}
	return true
}
