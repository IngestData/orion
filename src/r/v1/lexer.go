package v1

import (
	"strings"
	"unicode"
)

// Lexer turns R source code into a stream of Tokens.
type Lexer struct {
	input []rune

	pos     int // index of ch in input
	readPos int // index of next rune to read
	ch      rune

	line int
	col  int
}

// NewLexer creates a Lexer over the given R source.
func NewLexer(source string) *Lexer {
	l := &Lexer{input: []rune(source), line: 1, col: 0}
	l.readChar()
	return l
}

// Tokenize lexes the entire source and returns the resulting Tokens,
// terminated by a single EOF token.
func Tokenize(source string) []Token {
	l := NewLexer(source)
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

const nul = rune(0)

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = nul
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++

	if l.ch == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return nul
	}
	return l.input[l.readPos]
}

func (l *Lexer) peekCharAt(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.input) || idx < 0 {
		return nul
	}
	return l.input[idx]
}

// NextToken scans and returns the next Token in the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	line, col := l.line, l.col

	if l.ch == nul {
		return Token{Type: EOF, Literal: "", Line: line, Col: col}
	}

	if l.ch == '\n' {
		l.readChar()
		return Token{Type: NEWLINE, Literal: "\n", Line: line, Col: col}
	}

	switch {
	case (l.ch == 'r' || l.ch == 'R') && (l.peekChar() == '"' || l.peekChar() == '\''):
		return l.readRawString(line, col)
	case isIdentStart(l.ch, l.peekChar()):
		return l.readIdent(line, col)
	case l.ch == '`':
		return l.readBacktickIdent(line, col)
	case isDigit(l.ch), l.ch == '.' && isDigit(l.peekChar()):
		return l.readNumber(line, col)
	case l.ch == '"' || l.ch == '\'':
		return l.readString(l.ch, line, col)
	}

	return l.readOperator(line, col)
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		switch {
		case l.ch == ' ' || l.ch == '\t' || l.ch == '\r':
			l.readChar()
		case l.ch == '#':
			for l.ch != '\n' && l.ch != nul {
				l.readChar()
			}
		default:
			return
		}
	}
}

func isDigit(ch rune) bool { return ch >= '0' && ch <= '9' }

func isHexDigit(ch rune) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isLetter(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentPart(ch rune) bool {
	return isLetter(ch) || isDigit(ch) || ch == '.'
}

// isIdentStart reports whether ch can begin an identifier. A leading '.' is
// only an identifier start if it is not immediately followed by a digit
// (".5" is a number literal, ".x" is the identifier ".x").
func isIdentStart(ch, next rune) bool {
	if isLetter(ch) {
		return true
	}
	if ch == '.' && !isDigit(next) {
		return true
	}
	return false
}

func (l *Lexer) readIdent(line, col int) Token {
	start := l.pos
	for isIdentPart(l.ch) {
		l.readChar()
	}
	lit := string(l.input[start:l.pos])
	return Token{Type: LookupIdent(lit), Literal: lit, Line: line, Col: col}
}

func (l *Lexer) readBacktickIdent(line, col int) Token {
	l.readChar() // consume opening `
	start := l.pos
	for l.ch != '`' && l.ch != nul {
		l.readChar()
	}
	lit := string(l.input[start:l.pos])
	if l.ch == '`' {
		l.readChar() // consume closing `
	}
	return Token{Type: IDENT, Literal: lit, Line: line, Col: col}
}

func (l *Lexer) readNumber(line, col int) Token {
	start := l.pos

	if l.ch == '0' && (l.peekChar() == 'x' || l.peekChar() == 'X') {
		l.readChar()
		l.readChar()
		for isHexDigit(l.ch) {
			l.readChar()
		}
	} else {
		for isDigit(l.ch) {
			l.readChar()
		}
		if l.ch == '.' {
			l.readChar()
			for isDigit(l.ch) {
				l.readChar()
			}
		}
		if l.ch == 'e' || l.ch == 'E' {
			l.readChar()
			if l.ch == '+' || l.ch == '-' {
				l.readChar()
			}
			for isDigit(l.ch) {
				l.readChar()
			}
		}
	}

	// Integer ("L") or complex ("i") suffix.
	if l.ch == 'L' || l.ch == 'i' {
		l.readChar()
	}

	lit := string(l.input[start:l.pos])
	return Token{Type: NUMBER, Literal: lit, Line: line, Col: col}
}

var escapeReplacer = strings.NewReplacer(
	`\n`, "\n",
	`\t`, "\t",
	`\r`, "\r",
	`\\`, "\\",
	`\"`, "\"",
	`\'`, "'",
	"\\`", "`",
	`\0`, "\x00",
)

func (l *Lexer) readString(quote rune, line, col int) Token {
	l.readChar() // consume opening quote
	var sb strings.Builder
	for l.ch != quote && l.ch != nul {
		if l.ch == '\\' && l.peekChar() != nul {
			sb.WriteRune(l.ch)
			l.readChar()
			sb.WriteRune(l.ch)
			l.readChar()
			continue
		}
		sb.WriteRune(l.ch)
		l.readChar()
	}
	if l.ch == quote {
		l.readChar() // consume closing quote
	}
	return Token{Type: STRING, Literal: escapeReplacer.Replace(sb.String()), Line: line, Col: col}
}

// readRawString scans an R raw string literal: r"(...)"  R'[...]'  r"---{...}---"
// The opening is: r|R, a quote ('"' or '\”), zero or more '-', then one of
// ( [ {. The matching close is the mirrored bracket followed by the same
// number of dashes and the same quote character.
func (l *Lexer) readRawString(line, col int) Token {
	l.readChar() // consume 'r'/'R'
	quote := l.ch
	l.readChar() // consume quote

	dashes := 0
	for l.ch == '-' {
		dashes++
		l.readChar()
	}

	var open, closeCh rune
	switch l.ch {
	case '(':
		open, closeCh = '(', ')'
	case '[':
		open, closeCh = '[', ']'
	case '{':
		open, closeCh = '{', '}'
	default:
		// Malformed raw string; bail out gracefully.
		return Token{Type: ILLEGAL, Literal: string(l.ch), Line: line, Col: col}
	}
	l.readChar() // consume opening bracket
	_ = open

	closer := string(closeCh) + strings.Repeat("-", dashes) + string(quote)

	start := l.pos
	for l.ch != nul {
		if l.ch == closeCh && string(l.input[l.pos:min(l.pos+len(closer), len(l.input))]) == closer {
			break
		}
		l.readChar()
	}
	lit := string(l.input[start:l.pos])
	for i := 0; i < len(closer) && l.ch != nul; i++ {
		l.readChar()
	}
	return Token{Type: STRING, Literal: lit, Line: line, Col: col}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (l *Lexer) readOperator(line, col int) Token {
	ch := l.ch

	three := string(l.ch) + string(l.peekChar()) + string(l.peekCharAt(2))
	switch three {
	case "<<-":
		l.readChar()
		l.readChar()
		l.readChar()
		return Token{Type: SUPER_ASSIGN, Literal: three, Line: line, Col: col}
	case "->>":
		l.readChar()
		l.readChar()
		l.readChar()
		return Token{Type: RIGHT_SUPER_ASSIGN, Literal: three, Line: line, Col: col}
	case ":::":
		l.readChar()
		l.readChar()
		l.readChar()
		return Token{Type: TCOLON, Literal: three, Line: line, Col: col}
	}

	two := string(l.ch) + string(l.peekChar())
	switch two {
	case "<-":
		l.readChar()
		l.readChar()
		return Token{Type: ASSIGN, Literal: two, Line: line, Col: col}
	case "->":
		l.readChar()
		l.readChar()
		return Token{Type: RIGHT_ASSIGN, Literal: two, Line: line, Col: col}
	case "<=":
		l.readChar()
		l.readChar()
		return Token{Type: LE, Literal: two, Line: line, Col: col}
	case ">=":
		l.readChar()
		l.readChar()
		return Token{Type: GE, Literal: two, Line: line, Col: col}
	case "==":
		l.readChar()
		l.readChar()
		return Token{Type: EQ, Literal: two, Line: line, Col: col}
	case "!=":
		l.readChar()
		l.readChar()
		return Token{Type: NE, Literal: two, Line: line, Col: col}
	case "&&":
		l.readChar()
		l.readChar()
		return Token{Type: AND2, Literal: two, Line: line, Col: col}
	case "||":
		l.readChar()
		l.readChar()
		return Token{Type: OR2, Literal: two, Line: line, Col: col}
	case "::":
		l.readChar()
		l.readChar()
		return Token{Type: DCOLON, Literal: two, Line: line, Col: col}
	case "|>":
		l.readChar()
		l.readChar()
		return Token{Type: PIPE, Literal: two, Line: line, Col: col}
	}

	if ch == '%' {
		start := l.pos
		l.readChar()
		for l.ch != '%' && l.ch != nul && l.ch != '\n' {
			l.readChar()
		}
		if l.ch == '%' {
			l.readChar()
		}
		lit := string(l.input[start:l.pos])
		return Token{Type: SPECIAL, Literal: lit, Line: line, Col: col}
	}

	single := map[rune]Type{
		'(': LPAREN, ')': RPAREN,
		'{': LBRACE, '}': RBRACE,
		'[': LBRACKET, ']': RBRACKET,
		',': COMMA, ';': SEMICOLON,
		'+': PLUS, '-': MINUS, '*': STAR, '/': SLASH, '^': CARET,
		'<': LT, '>': GT,
		'!': NOT, '&': AND, '|': OR,
		'~': TILDE, '?': QUESTION, ':': COLON,
		'$': DOLLAR, '@': AT, '\\': BACKSLASH,
		'=': EQUAL_ASSIGN,
	}
	if t, ok := single[ch]; ok {
		l.readChar()
		return Token{Type: t, Literal: string(ch), Line: line, Col: col}
	}

	l.readChar()
	return Token{Type: ILLEGAL, Literal: string(ch), Line: line, Col: col}
}
