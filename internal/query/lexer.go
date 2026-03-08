package query

// TokenType represents a lexer token type.
type TokenType int

const (
	TokenEOF    TokenType = iota
	TokenAnd              // AND
	TokenOr               // OR
	TokenNot              // NOT
	TokenLParen           // (
	TokenRParen           // )
	TokenWord             // a bare word or quoted string
	TokenOp               // :, >, >=, <, <=, ~
)

// Token is a lexer token.
type Token struct {
	Type  TokenType
	Value string
}

// Lexer tokenizes a query string.
type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

// Lex tokenizes the input string.
func Lex(input string) []Token {
	l := &Lexer{input: input}
	l.scan()
	return l.tokens
}

func (l *Lexer) scan() {
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		ch := l.input[l.pos]

		switch {
		case ch == '(':
			l.tokens = append(l.tokens, Token{Type: TokenLParen, Value: "("})
			l.pos++
		case ch == ')':
			l.tokens = append(l.tokens, Token{Type: TokenRParen, Value: ")"})
			l.pos++
		case ch == '"':
			l.scanQuotedString()
		default:
			l.scanWord()
		}
	}
	l.tokens = append(l.tokens, Token{Type: TokenEOF})
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
		l.pos++
	}
}

func (l *Lexer) scanQuotedString() {
	l.pos++ // skip opening quote
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.pos++ // skip escaped char
		}
		l.pos++
	}
	value := l.input[start:l.pos]
	if l.pos < len(l.input) {
		l.pos++ // skip closing quote
	}
	l.tokens = append(l.tokens, Token{Type: TokenWord, Value: value})
}

func (l *Lexer) scanWord() {
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '(' || ch == ')' {
			break
		}
		// Check for operators within the word
		if l.pos > start {
			if ch == ':' || ch == '~' {
				// Emit the word before the operator
				l.tokens = append(l.tokens, Token{Type: TokenWord, Value: l.input[start:l.pos]})
				l.tokens = append(l.tokens, Token{Type: TokenOp, Value: string(ch)})
				l.pos++
				// Now read the value part
				l.scanValue()
				return
			}
			if ch == '>' || ch == '<' {
				l.tokens = append(l.tokens, Token{Type: TokenWord, Value: l.input[start:l.pos]})
				if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
					l.tokens = append(l.tokens, Token{Type: TokenOp, Value: l.input[l.pos : l.pos+2]})
					l.pos += 2
				} else {
					l.tokens = append(l.tokens, Token{Type: TokenOp, Value: string(ch)})
					l.pos++
				}
				l.scanValue()
				return
			}
		}
		l.pos++
	}

	word := l.input[start:l.pos]
	if word == "" {
		return
	}

	switch word {
	case "AND":
		l.tokens = append(l.tokens, Token{Type: TokenAnd, Value: word})
	case "OR":
		l.tokens = append(l.tokens, Token{Type: TokenOr, Value: word})
	case "NOT":
		l.tokens = append(l.tokens, Token{Type: TokenNot, Value: word})
	default:
		l.tokens = append(l.tokens, Token{Type: TokenWord, Value: word})
	}
}

func (l *Lexer) scanValue() {
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		l.tokens = append(l.tokens, Token{Type: TokenWord, Value: ""})
		return
	}

	if l.input[l.pos] == '"' {
		l.scanQuotedString()
		return
	}

	// Unquoted value: read until whitespace or paren
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '(' || ch == ')' {
			break
		}
		l.pos++
	}
	l.tokens = append(l.tokens, Token{Type: TokenWord, Value: l.input[start:l.pos]})
}
