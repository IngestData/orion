package v1

// Node is implemented by every AST node produced by the Parser.
type Node interface {
	node()
}

// Program is the root of a parsed R source file: a sequence of top-level
// expressions/statements.
type Program struct {
	Body []Node
}

// Ident is a bare name, e.g. x, .hidden, ..., ..1, or a backtick-quoted name.
type Ident struct {
	Name string
}

// NumberLit is a numeric literal, kept as its original source text
// (e.g. "10L", "0x1A", "1e-3", "2i").
type NumberLit struct {
	Value string
}

// StringLit is a string literal with escape sequences already resolved.
type StringLit struct {
	Value string
}

// ConstLit covers the reserved constants: TRUE, FALSE, NULL, NA (and its
// typed variants), Inf and NaN. Kind is the lexer Token Type that produced
// it (TRUE, FALSE, NULL, NA, INF or NAN).
type ConstLit struct {
	Kind    Type
	Literal string
}

// UnaryExpr is a prefix operator applied to a single operand, e.g. -x, !x,
// +x, or a one-sided formula ~x.
type UnaryExpr struct {
	Op Type
	X  Node
}

// BinaryExpr is an infix operator applied to two operands, e.g. a + b,
// a %in% b, a:b, a && b, a ~ b.
type BinaryExpr struct {
	Op   Type
	X, Y Node
}

// AssignExpr covers all five assignment forms: <-, <<-, =, ->, ->>.
// Lhs/Rhs are stored in source order regardless of which side the value
// flows from; Op distinguishes direction (-> / ->> assign Lhs into Rhs).
type AssignExpr struct {
	Op  Type
	Lhs Node
	Rhs Node
}

// Arg is one call/index argument. Name is empty for positional arguments.
// Value is nil for a deliberately missing/empty argument (e.g. the middle
// slot in df[, 1]).
type Arg struct {
	Name  string
	Value Node
}

// CallExpr is a function call, e.g. f(x, y = 1).
type CallExpr struct {
	Fun  Node
	Args []Arg
}

// IndexExpr is subsetting via [ ] or [[ ]].
type IndexExpr struct {
	X      Node
	Args   []Arg
	Double bool // true for [[ ]]
}

// DollarExpr is x$name member access.
type DollarExpr struct {
	X    Node
	Name string
}

// AtExpr is x@name slot access.
type AtExpr struct {
	X    Node
	Name string
}

// NamespaceExpr is pkg::name or pkg:::name.
type NamespaceExpr struct {
	Pkg      string
	Name     string
	Internal bool // true for :::
}

// Param is a single formal parameter in a function definition. Default is
// nil when the parameter has no default value.
type Param struct {
	Name    string
	Default Node
}

// FunctionDef is a function definition, from either `function(...) body`
// or the `\(...) body` shorthand.
type FunctionDef struct {
	Params []Param
	Body   Node
}

// IfExpr is `if (Cond) Then [else Else]`. Else is nil when absent.
type IfExpr struct {
	Cond Node
	Then Node
	Else Node
}

// ForExpr is `for (Var in Seq) Body`.
type ForExpr struct {
	Var  string
	Seq  Node
	Body Node
}

// WhileExpr is `while (Cond) Body`.
type WhileExpr struct {
	Cond Node
	Body Node
}

// RepeatExpr is `repeat Body`.
type RepeatExpr struct {
	Body Node
}

// BlockExpr is a `{ ... }` block of statements.
type BlockExpr struct {
	Stmts []Node
}

// BreakStmt is `break`.
type BreakStmt struct{}

// NextStmt is `next`.
type NextStmt struct{}

func (*Program) node()       {}
func (*Ident) node()         {}
func (*NumberLit) node()     {}
func (*StringLit) node()     {}
func (*ConstLit) node()      {}
func (*UnaryExpr) node()     {}
func (*BinaryExpr) node()    {}
func (*AssignExpr) node()    {}
func (*CallExpr) node()      {}
func (*IndexExpr) node()     {}
func (*DollarExpr) node()    {}
func (*AtExpr) node()        {}
func (*NamespaceExpr) node() {}
func (*FunctionDef) node()   {}
func (*IfExpr) node()        {}
func (*ForExpr) node()       {}
func (*WhileExpr) node()     {}
func (*RepeatExpr) node()    {}
func (*BlockExpr) node()     {}
func (*BreakStmt) node()     {}
func (*NextStmt) node()      {}
