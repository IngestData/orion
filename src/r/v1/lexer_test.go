package v1

import "testing"

func TestNextTokenBasics(t *testing.T) {
	source := `x <- 1 + 2 * (3 - 4) / 5
y <<- TRUE && FALSE || !NULL
f <- function(a, b = 1) { a + b }
name$field@slot
pkg::fun(x %in% y, z[[1]], w[1, ])
"hello\nworld" 'it\'s' r"(raw\nstring)"
0x1AL 10L 3.14 1e-3 2i
# a comment
x |> f()
`
	want := []Token{
		{Type: IDENT, Literal: "x"},
		{Type: ASSIGN, Literal: "<-"},
		{Type: NUMBER, Literal: "1"},
		{Type: PLUS, Literal: "+"},
		{Type: NUMBER, Literal: "2"},
		{Type: STAR, Literal: "*"},
		{Type: LPAREN, Literal: "("},
		{Type: NUMBER, Literal: "3"},
		{Type: MINUS, Literal: "-"},
		{Type: NUMBER, Literal: "4"},
		{Type: RPAREN, Literal: ")"},
		{Type: SLASH, Literal: "/"},
		{Type: NUMBER, Literal: "5"},
		{Type: NEWLINE, Literal: "\n"},

		{Type: IDENT, Literal: "y"},
		{Type: SUPER_ASSIGN, Literal: "<<-"},
		{Type: TRUE, Literal: "TRUE"},
		{Type: AND2, Literal: "&&"},
		{Type: FALSE, Literal: "FALSE"},
		{Type: OR2, Literal: "||"},
		{Type: NOT, Literal: "!"},
		{Type: NULL, Literal: "NULL"},
		{Type: NEWLINE, Literal: "\n"},

		{Type: IDENT, Literal: "f"},
		{Type: ASSIGN, Literal: "<-"},
		{Type: FUNCTION, Literal: "function"},
		{Type: LPAREN, Literal: "("},
		{Type: IDENT, Literal: "a"},
		{Type: COMMA, Literal: ","},
		{Type: IDENT, Literal: "b"},
		{Type: EQUAL_ASSIGN, Literal: "="},
		{Type: NUMBER, Literal: "1"},
		{Type: RPAREN, Literal: ")"},
		{Type: LBRACE, Literal: "{"},
		{Type: IDENT, Literal: "a"},
		{Type: PLUS, Literal: "+"},
		{Type: IDENT, Literal: "b"},
		{Type: RBRACE, Literal: "}"},
		{Type: NEWLINE, Literal: "\n"},

		{Type: IDENT, Literal: "name"},
		{Type: DOLLAR, Literal: "$"},
		{Type: IDENT, Literal: "field"},
		{Type: AT, Literal: "@"},
		{Type: IDENT, Literal: "slot"},
		{Type: NEWLINE, Literal: "\n"},

		{Type: IDENT, Literal: "pkg"},
		{Type: DCOLON, Literal: "::"},
		{Type: IDENT, Literal: "fun"},
		{Type: LPAREN, Literal: "("},
		{Type: IDENT, Literal: "x"},
		{Type: SPECIAL, Literal: "%in%"},
		{Type: IDENT, Literal: "y"},
		{Type: COMMA, Literal: ","},
		{Type: IDENT, Literal: "z"},
		{Type: LBRACKET, Literal: "["},
		{Type: LBRACKET, Literal: "["},
		{Type: NUMBER, Literal: "1"},
		{Type: RBRACKET, Literal: "]"},
		{Type: RBRACKET, Literal: "]"},
		{Type: COMMA, Literal: ","},
		{Type: IDENT, Literal: "w"},
		{Type: LBRACKET, Literal: "["},
		{Type: NUMBER, Literal: "1"},
		{Type: COMMA, Literal: ","},
		{Type: RBRACKET, Literal: "]"},
		{Type: RPAREN, Literal: ")"},
		{Type: NEWLINE, Literal: "\n"},

		{Type: STRING, Literal: "hello\nworld"},
		{Type: STRING, Literal: "it's"},
		{Type: STRING, Literal: `raw\nstring`},
		{Type: NEWLINE, Literal: "\n"},

		{Type: NUMBER, Literal: "0x1AL"},
		{Type: NUMBER, Literal: "10L"},
		{Type: NUMBER, Literal: "3.14"},
		{Type: NUMBER, Literal: "1e-3"},
		{Type: NUMBER, Literal: "2i"},
		{Type: NEWLINE, Literal: "\n"},
		{Type: NEWLINE, Literal: "\n"}, // end of the comment-only line

		{Type: IDENT, Literal: "x"},
		{Type: PIPE, Literal: "|>"},
		{Type: IDENT, Literal: "f"},
		{Type: LPAREN, Literal: "("},
		{Type: RPAREN, Literal: ")"},
		{Type: NEWLINE, Literal: "\n"},

		{Type: EOF, Literal: ""},
	}

	got := Tokenize(source)
	if len(got) != len(want) {
		t.Fatalf("token count mismatch: got %d, want %d\ngot: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Type != w.Type || g.Literal != w.Literal {
			t.Errorf("token %d: got {%s %q}, want {%s %q}", i, g.Type, g.Literal, w.Type, w.Literal)
		}
	}
}

func TestIdentifiersAndDots(t *testing.T) {
	cases := []struct {
		src  string
		want Token
	}{
		{".hidden", Token{Type: IDENT, Literal: ".hidden"}},
		{"...", Token{Type: IDENT, Literal: "..."}},
		{"..1", Token{Type: IDENT, Literal: "..1"}},
		{"my.var_2", Token{Type: IDENT, Literal: "my.var_2"}},
		{"`a weird name`", Token{Type: IDENT, Literal: "a weird name"}},
		{".5", Token{Type: NUMBER, Literal: ".5"}},
	}
	for _, c := range cases {
		toks := Tokenize(c.src)
		if len(toks) < 1 {
			t.Fatalf("no tokens for %q", c.src)
		}
		got := toks[0]
		if got.Type != c.want.Type || got.Literal != c.want.Literal {
			t.Errorf("Tokenize(%q)[0] = {%s %q}, want {%s %q}", c.src, got.Type, got.Literal, c.want.Type, c.want.Literal)
		}
	}
}

func TestComments(t *testing.T) {
	toks := Tokenize("x <- 1 # assign one\ny")
	var types []Type
	for _, tok := range toks {
		types = append(types, tok.Type)
	}
	want := []Type{IDENT, ASSIGN, NUMBER, NEWLINE, IDENT, EOF}
	if len(types) != len(want) {
		t.Fatalf("got %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Errorf("token %d: got %s, want %s", i, types[i], want[i])
		}
	}
}
