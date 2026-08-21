package v1

import (
	"fmt"
	"testing"
)

// dump renders a Node as a fully-parenthesized s-expression so tests can
// assert on shape/precedence without hand-building AST structs.
func dump(n Node) string {
	switch v := n.(type) {
	case nil:
		return "<nil>"
	case *Ident:
		return v.Name
	case *NumberLit:
		return v.Value
	case *StringLit:
		return fmt.Sprintf("%q", v.Value)
	case *ConstLit:
		return v.Literal
	case *UnaryExpr:
		return fmt.Sprintf("(%s %s)", v.Op, dump(v.X))
	case *BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", v.Op, dump(v.X), dump(v.Y))
	case *AssignExpr:
		return fmt.Sprintf("(%s %s %s)", v.Op, dump(v.Lhs), dump(v.Rhs))
	case *CallExpr:
		s := fmt.Sprintf("(call %s", dump(v.Fun))
		for _, a := range v.Args {
			if a.Name != "" {
				s += fmt.Sprintf(" %s=%s", a.Name, dump(a.Value))
			} else {
				s += " " + dump(a.Value)
			}
		}
		return s + ")"
	case *IndexExpr:
		br := "["
		if v.Double {
			br = "[["
		}
		s := fmt.Sprintf("(index%s %s", br, dump(v.X))
		for _, a := range v.Args {
			s += " " + dump(a.Value)
		}
		return s + ")"
	case *DollarExpr:
		return fmt.Sprintf("($ %s %s)", dump(v.X), v.Name)
	case *AtExpr:
		return fmt.Sprintf("(@ %s %s)", dump(v.X), v.Name)
	case *NamespaceExpr:
		op := "::"
		if v.Internal {
			op = ":::"
		}
		return fmt.Sprintf("(%s %s %s)", op, v.Pkg, v.Name)
	case *FunctionDef:
		s := "(function ("
		for i, p := range v.Params {
			if i > 0 {
				s += ", "
			}
			s += p.Name
			if p.Default != nil {
				s += "=" + dump(p.Default)
			}
		}
		return s + ") " + dump(v.Body) + ")"
	case *IfExpr:
		s := fmt.Sprintf("(if %s %s", dump(v.Cond), dump(v.Then))
		if v.Else != nil {
			s += " " + dump(v.Else)
		}
		return s + ")"
	case *ForExpr:
		return fmt.Sprintf("(for %s %s %s)", v.Var, dump(v.Seq), dump(v.Body))
	case *WhileExpr:
		return fmt.Sprintf("(while %s %s)", dump(v.Cond), dump(v.Body))
	case *RepeatExpr:
		return fmt.Sprintf("(repeat %s)", dump(v.Body))
	case *BlockExpr:
		s := "(block"
		for _, st := range v.Stmts {
			s += " " + dump(st)
		}
		return s + ")"
	case *BreakStmt:
		return "break"
	case *NextStmt:
		return "next"
	default:
		return fmt.Sprintf("<?%T>", n)
	}
}

func parseOneExpr(t *testing.T, src string) Node {
	t.Helper()
	prog, errs := Parse(src)
	if len(errs) != 0 {
		t.Fatalf("parse errors for %q: %v", src, errs)
	}
	if len(prog.Body) != 1 {
		t.Fatalf("expected exactly 1 statement for %q, got %d: %#v", src, len(prog.Body), prog.Body)
	}
	return prog.Body[0]
}

func TestPrecedenceAndAssociativity(t *testing.T) {
	cases := []struct{ src, want string }{
		{"1 + 2 * 3", "(+ 1 (* 2 3))"},
		{"(1 + 2) * 3", "(* (+ 1 2) 3)"},
		{"2 ^ 3 ^ 2", "(^ 2 (^ 3 2))"},
		{"-2 ^ 2", "(- (^ 2 2))"},
		{"-x[1]", "(- (index[ x 1))"},
		{"1:2 + 1", "(+ (: 1 2) 1)"},
		{"1 + 1:2", "(+ 1 (: 1 2))"},
		{"a && b || c", "(|| (&& a b) c)"},
		{"!a & b", "(& (! a) b)"},
		{"!a == b", "(! (== a b))"},
		{"a <- b <- 1", "(<- a (<- b 1))"},
		{"a = b = 1", "(= a (= b 1))"},
		{"x %in% y %in% z", "(%% (%% x y) z)"},
		{"x %in% 1:2", "(%% x (: 1 2))"},
		{"1 -> x -> y", "(-> 1 (-> x y))"},
		{"y ~ x + z", "(~ y (+ x z))"},
		{"a::b", "(:: a b)"},
		{"a:::b", "(::: a b)"},
	}
	for _, c := range cases {
		got := dump(parseOneExpr(t, c.src))
		if got != c.want {
			t.Errorf("dump(parse(%q)) = %s, want %s", c.src, got, c.want)
		}
	}
}

func TestCallsIndexingAndAccess(t *testing.T) {
	cases := []struct{ src, want string }{
		{"f(x, y = 1)", "(call f x y=1)"},
		{"f()", "(call f)"},
		{"x[1]", "(index[ x 1)"},
		{"x[[1]]", "(index[[ x 1)"},
		{"df[, 1]", "(index[ df <nil> 1)"},
		{"df[1, ]", "(index[ df 1 <nil>)"},
		{"obj$field", "($ obj field)"},
		{"obj@slot", "(@ obj slot)"},
		{"f(x)(y)", "(call (call f x) y)"},
		{"x[1][2]", "(index[ (index[ x 1) 2)"},
	}
	for _, c := range cases {
		got := dump(parseOneExpr(t, c.src))
		if got != c.want {
			t.Errorf("dump(parse(%q)) = %s, want %s", c.src, got, c.want)
		}
	}
}

func TestControlFlowAndFunctions(t *testing.T) {
	cases := []struct{ src, want string }{
		{"if (x > 0) 1 else -1", "(if (> x 0) 1 (- 1))"},
		{"if (x) {\n  1\n} else {\n  2\n}", "(if x (block 1) (block 2))"},
		{"for (i in 1:10) print(i)", "(for i (: 1 10) (call print i))"},
		{"while (x < 10) x <- x + 1", "(while (< x 10) (<- x (+ x 1)))"},
		{"repeat break", "(repeat break)"},
		{"function(a, b = 1) a + b", "(function (a, b=1) (+ a b))"},
		{"\\(x) x * 2", "(function (x) (* x 2))"},
	}
	for _, c := range cases {
		got := dump(parseOneExpr(t, c.src))
		if got != c.want {
			t.Errorf("dump(parse(%q)) = %s, want %s", c.src, got, c.want)
		}
	}
}

func TestIfElseAcrossNewlineInsideBlock(t *testing.T) {
	src := `
if (x > 0) {
  1
}
else {
  2
}
`
	prog, errs := Parse(src)
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(prog.Body) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Body))
	}
	got := dump(prog.Body[0])
	want := "(if (> x 0) (block 1) (block 2))"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestIfWithoutElseDoesNotSwallowNextStatement(t *testing.T) {
	src := "if (x > 0) 1\ny <- 2"
	prog, errs := Parse(src)
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(prog.Body) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(prog.Body), prog.Body)
	}
	if got := dump(prog.Body[0]); got != "(if (> x 0) 1)" {
		t.Errorf("stmt 0 = %s", got)
	}
	if got := dump(prog.Body[1]); got != "(<- y 2)" {
		t.Errorf("stmt 1 = %s", got)
	}
}

func TestMultiStatementProgram(t *testing.T) {
	src := `x <- 1
y <- 2
z <- x + y
`
	prog, errs := Parse(src)
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(prog.Body) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(prog.Body))
	}
	want := []string{"(<- x 1)", "(<- y 2)", "(<- z (+ x y))"}
	for i, w := range want {
		if got := dump(prog.Body[i]); got != w {
			t.Errorf("stmt %d = %s, want %s", i, got, w)
		}
	}
}

func TestLiteralsAndConstants(t *testing.T) {
	cases := []struct{ src, want string }{
		{"TRUE", "TRUE"},
		{"FALSE", "FALSE"},
		{"NULL", "NULL"},
		{"NA", "NA"},
		{"Inf", "Inf"},
		{"NaN", "NaN"},
		{`"a string"`, `"a string"`},
	}
	for _, c := range cases {
		got := dump(parseOneExpr(t, c.src))
		if got != c.want {
			t.Errorf("dump(parse(%q)) = %s, want %s", c.src, got, c.want)
		}
	}
}
