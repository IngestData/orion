package v1

import "fmt"

// Precedence levels, lowest to highest binding power. These mirror the
// operator precedence table in the R Language Definition (section on
// "Operator precedence").
const (
	LOWEST       int = iota
	HELP             // ?
	ASSIGN_EQ        // =
	ASSIGN_L         // <- <<-
	ASSIGN_R         // -> ->>
	TILDE_PREC       // ~
	OR_PREC          // | ||
	AND_PREC         // & &&
	NOT_PREC         // ! (prefix)
	COMPARE          // == != < > <= >=
	ADD              // binary + -
	MUL              // * /
	SPECIAL_PREC     // %any% |>
	RANGE            // :
	UNARY            // prefix - +
	POW              // ^
	ACCESS           // [ [[ ( $ @
	NAMESPACE        // :: :::
)

var precedences = map[Type]int{
	QUESTION:           HELP,
	EQUAL_ASSIGN:       ASSIGN_EQ,
	ASSIGN:             ASSIGN_L,
	SUPER_ASSIGN:       ASSIGN_L,
	RIGHT_ASSIGN:       ASSIGN_R,
	RIGHT_SUPER_ASSIGN: ASSIGN_R,
	TILDE:              TILDE_PREC,
	OR:                 OR_PREC,
	OR2:                OR_PREC,
	AND:                AND_PREC,
	AND2:               AND_PREC,
	EQ:                 COMPARE,
	NE:                 COMPARE,
	LT:                 COMPARE,
	GT:                 COMPARE,
	LE:                 COMPARE,
	GE:                 COMPARE,
	PLUS:               ADD,
	MINUS:              ADD,
	STAR:               MUL,
	SLASH:              MUL,
	SPECIAL:            SPECIAL_PREC,
	PIPE:               SPECIAL_PREC,
	COLON:              RANGE,
	CARET:              POW,
	LBRACKET:           ACCESS,
	LPAREN:             ACCESS,
	DOLLAR:             ACCESS,
	AT:                 ACCESS,
	DCOLON:             NAMESPACE,
	TCOLON:             NAMESPACE,
}

// rightAssoc is the set of infix operator precedence tiers that associate
// right-to-left.
var rightAssoc = map[int]bool{
	ASSIGN_EQ:  true,
	ASSIGN_L:   true,
	ASSIGN_R:   true,
	TILDE_PREC: true,
	POW:        true,
}

// Parser is a recursive-descent (Pratt) parser for R source code.
type Parser struct {
	l *Lexer

	curToken  Token
	peekToken Token

	// depthStack tracks ( / [ nesting so newlines can be skipped while an
	// expression is unfinished. A new scope is pushed on '{' and popped on
	// '}', because newlines inside a block are significant regardless of
	// how many parens enclose the block itself.
	depthStack []int

	errors []string
}

// NewParser builds a Parser over the given Lexer.
func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l, depthStack: []int{0}}
	p.advance()
	p.advance()
	return p
}

// Parse lexes+parses R source text directly into a Program.
func Parse(source string) (*Program, []string) {
	p := NewParser(NewLexer(source))
	return p.ParseProgram(), p.errors
}

// Errors returns any parse errors accumulated so far.
func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf("line %d: %s", p.curToken.Line, fmt.Sprintf(format, args...)))
}

func (p *Parser) currentDepth() int { return p.depthStack[len(p.depthStack)-1] }

func (p *Parser) advance() {
	p.curToken = p.peekToken
	switch p.curToken.Type {
	case LPAREN, LBRACKET:
		p.depthStack[len(p.depthStack)-1]++
	case RPAREN, RBRACKET:
		if p.depthStack[len(p.depthStack)-1] > 0 {
			p.depthStack[len(p.depthStack)-1]--
		}
	case LBRACE:
		p.depthStack = append(p.depthStack, 0)
	case RBRACE:
		if len(p.depthStack) > 1 {
			p.depthStack = p.depthStack[:len(p.depthStack)-1]
		}
	}
	p.peekToken = p.readSignificant()
}

func (p *Parser) readSignificant() Token {
	for {
		t := p.l.NextToken()
		if t.Type == NEWLINE && p.currentDepth() > 0 {
			continue
		}
		return t
	}
}

func (p *Parser) curIs(t Type) bool  { return p.curToken.Type == t }
func (p *Parser) peekIs(t Type) bool { return p.peekToken.Type == t }

func (p *Parser) expect(t Type) bool {
	if p.peekIs(t) {
		p.advance()
		return true
	}
	p.errorf("expected next token to be %s, got %s (%q) instead", t, p.peekToken.Type, p.peekToken.Literal)
	return false
}

func (p *Parser) peekPrecedence() int {
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return LOWEST
}

// skipSeparators consumes any run of NEWLINE/SEMICOLON tokens.
func (p *Parser) skipSeparators() {
	for p.curIs(NEWLINE) || p.curIs(SEMICOLON) {
		p.advance()
	}
}

// ParseProgram parses the whole token stream into a Program.
func (p *Parser) ParseProgram() *Program {
	prog := &Program{}
	p.skipSeparators()
	for !p.curIs(EOF) {
		stmt := p.parseExpression(LOWEST)
		if stmt != nil {
			prog.Body = append(prog.Body, stmt)
		}
		if !p.peekIs(NEWLINE) && !p.peekIs(SEMICOLON) && !p.peekIs(EOF) {
			p.errorf("unexpected token %s (%q); expected a newline or ;", p.peekToken.Type, p.peekToken.Literal)
		}
		p.advance()
		p.skipSeparators()
	}
	return prog
}

// parseExpression is the Pratt-parser core: parse a prefix expression, then
// keep absorbing infix/postfix operators whose precedence exceeds minPrec.
// On return, curToken is the last token that belongs to the expression;
// callers advance past it themselves.
func (p *Parser) parseExpression(minPrec int) Node {
	left := p.parsePrefix()
	if left == nil {
		return nil
	}

	for !p.peekIs(NEWLINE) && !p.peekIs(SEMICOLON) && !p.peekIs(EOF) && minPrec < p.peekPrecedence() {
		p.advance()
		left = p.parseInfix(left)
	}
	return left
}

func (p *Parser) parsePrefix() Node {
	switch p.curToken.Type {
	case IDENT:
		return &Ident{Name: p.curToken.Literal}
	case NUMBER:
		return &NumberLit{Value: p.curToken.Literal}
	case STRING:
		return &StringLit{Value: p.curToken.Literal}
	case TRUE, FALSE, NULL, NA, INF, NAN:
		return &ConstLit{Kind: p.curToken.Type, Literal: p.curToken.Literal}
	case MINUS, PLUS, NOT:
		return p.parseUnary()
	case TILDE:
		return p.parseTildePrefix()
	case QUESTION:
		return p.parseHelpPrefix()
	case LPAREN:
		return p.parseGroupedExpr()
	case LBRACE:
		return p.parseBlock()
	case IF:
		return p.parseIf()
	case FOR:
		return p.parseFor()
	case WHILE:
		return p.parseWhile()
	case REPEAT:
		return p.parseRepeat()
	case FUNCTION, BACKSLASH:
		return p.parseFunctionDef()
	case BREAK:
		return &BreakStmt{}
	case NEXT:
		return &NextStmt{}
	default:
		p.errorf("unexpected token %s (%q); expected start of expression", p.curToken.Type, p.curToken.Literal)
		return nil
	}
}

func (p *Parser) parseUnary() Node {
	op := p.curToken.Type
	prec := UNARY
	if op == NOT {
		prec = NOT_PREC
	}
	p.advance()
	x := p.parseExpression(prec)
	return &UnaryExpr{Op: op, X: x}
}

func (p *Parser) parseTildePrefix() Node {
	p.advance()
	x := p.parseExpression(TILDE_PREC)
	return &UnaryExpr{Op: TILDE, X: x}
}

func (p *Parser) parseHelpPrefix() Node {
	p.advance()
	x := p.parseExpression(HELP)
	return &UnaryExpr{Op: QUESTION, X: x}
}

func (p *Parser) parseGroupedExpr() Node {
	p.advance() // consume '(', land on first token of inner expr
	inner := p.parseExpression(LOWEST)
	if !p.expect(RPAREN) {
		return inner
	}
	return inner
}

func (p *Parser) parseBlock() Node {
	p.advance() // consume '{'
	block := &BlockExpr{}
	p.skipSeparators()
	for !p.curIs(RBRACE) && !p.curIs(EOF) {
		stmt := p.parseExpression(LOWEST)
		if stmt != nil {
			block.Stmts = append(block.Stmts, stmt)
		}
		if !p.peekIs(NEWLINE) && !p.peekIs(SEMICOLON) && !p.peekIs(RBRACE) && !p.peekIs(EOF) {
			p.errorf("unexpected token %s (%q) in block", p.peekToken.Type, p.peekToken.Literal)
		}
		p.advance()
		p.skipSeparators()
	}
	if !p.curIs(RBRACE) {
		p.errorf("expected } to close block, got %s", p.curToken.Type)
	}
	return block
}

func (p *Parser) parseIf() Node {
	if !p.expect(LPAREN) {
		return nil
	}
	p.advance()
	cond := p.parseExpression(LOWEST)
	if !p.expect(RPAREN) {
		return nil
	}
	p.advance()
	then := p.parseExpression(LOWEST)

	expr := &IfExpr{Cond: cond, Then: then}

	// `else` may follow after intervening separators (e.g. a closing brace
	// on its own line). Snapshot lexer + parser state so we can rewind if
	// no `else` turns out to follow -- the separator we're peeking past
	// belongs to the *enclosing* statement in that case.
	lexerSnapshot := *p.l
	curSnapshot, peekSnapshot := p.curToken, p.peekToken
	depthSnapshot := append([]int(nil), p.depthStack...)

	p.advance()
	p.skipSeparators()
	if p.curIs(ELSE) {
		p.advance()
		expr.Else = p.parseExpression(LOWEST)
	} else {
		*p.l = lexerSnapshot
		p.curToken, p.peekToken = curSnapshot, peekSnapshot
		p.depthStack = depthSnapshot
	}
	return expr
}

func (p *Parser) parseFor() Node {
	if !p.expect(LPAREN) {
		return nil
	}
	if !p.expect(IDENT) {
		return nil
	}
	name := p.curToken.Literal
	if !p.expect(IN) {
		return nil
	}
	p.advance()
	seq := p.parseExpression(LOWEST)
	if !p.expect(RPAREN) {
		return nil
	}
	p.advance()
	body := p.parseExpression(LOWEST)
	return &ForExpr{Var: name, Seq: seq, Body: body}
}

func (p *Parser) parseWhile() Node {
	if !p.expect(LPAREN) {
		return nil
	}
	p.advance()
	cond := p.parseExpression(LOWEST)
	if !p.expect(RPAREN) {
		return nil
	}
	p.advance()
	body := p.parseExpression(LOWEST)
	return &WhileExpr{Cond: cond, Body: body}
}

func (p *Parser) parseRepeat() Node {
	p.advance()
	body := p.parseExpression(LOWEST)
	return &RepeatExpr{Body: body}
}

func (p *Parser) parseFunctionDef() Node {
	if !p.expect(LPAREN) {
		return nil
	}
	p.advance() // move onto first param token (or ')')
	var params []Param
	for !p.curIs(RPAREN) && !p.curIs(EOF) {
		if !p.curIs(IDENT) {
			p.errorf("expected parameter name, got %s (%q)", p.curToken.Type, p.curToken.Literal)
			break
		}
		param := Param{Name: p.curToken.Literal}
		if p.peekIs(EQUAL_ASSIGN) {
			p.advance() // '='
			p.advance() // first token of default
			param.Default = p.parseExpression(LOWEST)
		}
		params = append(params, param)
		p.advance()
		if p.curIs(COMMA) {
			p.advance()
		}
	}
	if !p.curIs(RPAREN) {
		p.errorf("expected ) to close parameter list, got %s", p.curToken.Type)
		return &FunctionDef{Params: params}
	}
	p.advance() // move past ')' onto body
	body := p.parseExpression(LOWEST)
	return &FunctionDef{Params: params, Body: body}
}

// parseInfix dispatches on the operator that was just made curToken by the
// Pratt loop in parseExpression.
func (p *Parser) parseInfix(left Node) Node {
	switch p.curToken.Type {
	case LPAREN:
		return p.parseCall(left)
	case LBRACKET:
		return p.parseIndex(left)
	case DOLLAR:
		return p.parseDollar(left)
	case AT:
		return p.parseAt(left)
	case DCOLON, TCOLON:
		return p.parseNamespace(left)
	case ASSIGN, SUPER_ASSIGN, EQUAL_ASSIGN, RIGHT_ASSIGN, RIGHT_SUPER_ASSIGN:
		return p.parseAssign(left)
	default:
		return p.parseBinary(left)
	}
}

func (p *Parser) parseBinary(left Node) Node {
	op := p.curToken.Type
	prec := precedences[op]
	p.advance()
	nextMin := prec
	if rightAssoc[prec] {
		nextMin = prec - 1
	}
	right := p.parseExpression(nextMin)
	return &BinaryExpr{Op: op, X: left, Y: right}
}

func (p *Parser) parseAssign(left Node) Node {
	op := p.curToken.Type
	prec := precedences[op]
	p.advance()
	nextMin := prec
	if rightAssoc[prec] {
		nextMin = prec - 1
	}
	right := p.parseExpression(nextMin)
	return &AssignExpr{Op: op, Lhs: left, Rhs: right}
}

func (p *Parser) parseDollar(left Node) Node {
	switch {
	case p.peekIs(IDENT), p.peekIs(STRING):
		p.advance()
	default:
		p.errorf("expected name after $, got %s (%q)", p.peekToken.Type, p.peekToken.Literal)
		return left
	}
	return &DollarExpr{X: left, Name: p.curToken.Literal}
}

func (p *Parser) parseAt(left Node) Node {
	switch {
	case p.peekIs(IDENT), p.peekIs(STRING):
		p.advance()
	default:
		p.errorf("expected name after @, got %s (%q)", p.peekToken.Type, p.peekToken.Literal)
		return left
	}
	return &AtExpr{X: left, Name: p.curToken.Literal}
}

func (p *Parser) parseNamespace(left Node) Node {
	internal := p.curToken.Type == TCOLON
	pkgName := ""
	if pkg, ok := left.(*Ident); ok {
		pkgName = pkg.Name
	}
	if !p.expect(IDENT) {
		return left
	}
	return &NamespaceExpr{Pkg: pkgName, Name: p.curToken.Literal, Internal: internal}
}

func (p *Parser) parseCall(left Node) Node {
	args := p.parseArgList(RPAREN, true)
	return &CallExpr{Fun: left, Args: args}
}

func (p *Parser) parseIndex(left Node) Node {
	double := false
	if p.peekIs(LBRACKET) {
		double = true
		p.advance()
	}
	args := p.parseArgList(RBRACKET, false)
	if double && !p.expect(RBRACKET) {
		return &IndexExpr{X: left, Args: args, Double: true}
	}
	return &IndexExpr{X: left, Args: args, Double: double}
}

// parseArgList parses a comma-separated argument/index list. curToken is
// the opening delimiter on entry; curToken is the closing delimiter on
// exit. When requireValue is false, empty slots (e.g. the middle argument
// of df[, 1]) are recorded as an Arg with a nil Value; when true (function
// calls), empty slots are simply skipped.
func (p *Parser) parseArgList(closer Type, requireValue bool) []Arg {
	var args []Arg
	p.advance() // move past opening delimiter

	if p.curIs(closer) {
		return args
	}

	for {
		if p.curIs(COMMA) {
			if !requireValue {
				args = append(args, Arg{})
			}
		} else {
			var arg Arg
			if (p.curIs(IDENT) || p.curIs(STRING)) && p.peekIs(EQUAL_ASSIGN) {
				arg.Name = p.curToken.Literal
				p.advance() // consume name
				p.advance() // consume '='
				if !p.curIs(COMMA) && !p.curIs(closer) {
					arg.Value = p.parseExpression(LOWEST)
					p.advance()
				}
			} else {
				arg.Value = p.parseExpression(LOWEST)
				p.advance()
			}
			args = append(args, arg)
		}

		if p.curIs(closer) || p.curIs(EOF) {
			break
		}
		if p.curIs(COMMA) {
			p.advance()
			if p.curIs(closer) {
				if !requireValue {
					args = append(args, Arg{})
				}
				break
			}
			continue
		}

		p.errorf("unexpected token %s (%q) in argument list", p.curToken.Type, p.curToken.Literal)
		break
	}
	return args
}
