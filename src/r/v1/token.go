package v1

// Type identifies the lexical class of a Token.
type Type string

const (
	ILLEGAL Type = "ILLEGAL"
	EOF     Type = "EOF"
	NEWLINE Type = "NEWLINE"

	IDENT  Type = "IDENT"
	NUMBER Type = "NUMBER"
	STRING Type = "STRING"

	// Keywords / reserved constants.
	IF       Type = "IF"
	ELSE     Type = "ELSE"
	REPEAT   Type = "REPEAT"
	WHILE    Type = "WHILE"
	FUNCTION Type = "FUNCTION"
	FOR      Type = "FOR"
	IN       Type = "IN"
	NEXT     Type = "NEXT"
	BREAK    Type = "BREAK"
	TRUE     Type = "TRUE"
	FALSE    Type = "FALSE"
	NULL     Type = "NULL"
	NA       Type = "NA"
	INF      Type = "INF"
	NAN      Type = "NAN"

	// Delimiters.
	LPAREN    Type = "("
	RPAREN    Type = ")"
	LBRACE    Type = "{"
	RBRACE    Type = "}"
	LBRACKET  Type = "["
	RBRACKET  Type = "]"
	COMMA     Type = ","
	SEMICOLON Type = ";"

	// Operators.
	PLUS               Type = "+"
	MINUS              Type = "-"
	STAR               Type = "*"
	SLASH              Type = "/"
	CARET              Type = "^"
	SPECIAL            Type = "%%" // %any% infix operator, e.g. %%, %/%, %in%, %*%, user-defined
	LT                 Type = "<"
	GT                 Type = ">"
	LE                 Type = "<="
	GE                 Type = ">="
	EQ                 Type = "=="
	NE                 Type = "!="
	NOT                Type = "!"
	AND                Type = "&"
	AND2               Type = "&&"
	OR                 Type = "|"
	OR2                Type = "||"
	TILDE              Type = "~"
	QUESTION           Type = "?"
	COLON              Type = ":"
	DCOLON             Type = "::"
	TCOLON             Type = ":::"
	DOLLAR             Type = "$"
	AT                 Type = "@"
	BACKSLASH          Type = "\\"
	PIPE               Type = "|>"
	ASSIGN             Type = "<-"
	SUPER_ASSIGN       Type = "<<-"
	EQUAL_ASSIGN       Type = "="
	RIGHT_ASSIGN       Type = "->"
	RIGHT_SUPER_ASSIGN Type = "->>"
)

// keywords maps reserved words to their Token Type.
var keywords = map[string]Type{
	"if":            IF,
	"else":          ELSE,
	"repeat":        REPEAT,
	"while":         WHILE,
	"function":      FUNCTION,
	"for":           FOR,
	"in":            IN,
	"next":          NEXT,
	"break":         BREAK,
	"TRUE":          TRUE,
	"T":             TRUE,
	"FALSE":         FALSE,
	"F":             FALSE,
	"NULL":          NULL,
	"NA":            NA,
	"NA_integer_":   NA,
	"NA_real_":      NA,
	"NA_character_": NA,
	"NA_complex_":   NA,
	"Inf":           INF,
	"NaN":           NAN,
}

// LookupIdent returns the keyword Type for an identifier, or IDENT if it is
// not a reserved word.
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// Token is a single lexical token produced by the Lexer.
type Token struct {
	Type    Type
	Literal string
	Line    int
	Col     int
}
