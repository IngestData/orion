package v1

import (
	"os"
	"testing"
)

// TestParseSampleTransformer exercises the lexer+parser prototype against a
// realistic (if small) R transformer script, the R analogue of
// src/input/transformer.cs.
func TestParseSampleTransformer(t *testing.T) {
	src, err := os.ReadFile("../../input/transformer.R")
	if err != nil {
		t.Fatalf("reading sample: %v", err)
	}

	prog, errs := Parse(string(src))
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(prog.Body) != 4 {
		t.Fatalf("expected 4 top-level statements (library call + 3 assignments), got %d", len(prog.Body))
	}

	assign, ok := prog.Body[1].(*AssignExpr)
	if !ok {
		t.Fatalf("expected statement 1 to be an assignment, got %T", prog.Body[1])
	}
	fn, ok := assign.Rhs.(*FunctionDef)
	if !ok {
		t.Fatalf("expected Transformer <- function(...) ..., got %T", assign.Rhs)
	}
	if len(fn.Params) != 4 {
		t.Errorf("expected 4 params for Transformer(), got %d", len(fn.Params))
	}
}
