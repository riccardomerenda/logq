package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// isTimestampField returns true if the field name refers to a timestamp.
var timestampFields = map[string]bool{
	"timestamp": true, "ts": true, "time": true, "@timestamp": true, "datetime": true, "t": true,
}

// durationRe matches relative time values like "5m", "1h", "30s", "2d".
var durationRe = regexp.MustCompile(`^(\d+)([smhd])$`)

// Parser is a recursive descent parser for query expressions.
type Parser struct {
	tokens []Token
	pos    int
}

// ParseQuery parses a query string into an AST.
// Returns a MatchAll node for empty queries.
func ParseQuery(input string) (*Node, error) {
	tokens := Lex(input)
	p := &Parser{tokens: tokens}

	if p.peek().Type == TokenEOF {
		return &Node{Type: NodeMatchAll}, nil
	}

	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if p.peek().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token: %q", p.peek().Value)
	}

	return node, nil
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	t := p.peek()
	p.pos++
	return t
}

// parseOr → parseAnd ("OR" parseAnd)*
func (p *Parser) parseOr() (*Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TokenOr {
		p.next() // consume OR
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &Node{Type: NodeOr, Left: left, Right: right}
	}

	return left, nil
}

// parseAnd → parseUnary ("AND" parseUnary)*
func (p *Parser) parseAnd() (*Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TokenAnd {
		p.next() // consume AND
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &Node{Type: NodeAnd, Left: left, Right: right}
	}

	return left, nil
}

// parseUnary → "NOT" parseUnary | parsePrimary
func (p *Parser) parseUnary() (*Node, error) {
	if p.peek().Type == TokenNot {
		p.next() // consume NOT
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Node{Type: NodeNot, Child: child}, nil
	}

	return p.parsePrimary()
}

// parsePrimary → "(" parseOr ")" | field_op_value | fulltext
func (p *Parser) parsePrimary() (*Node, error) {
	if p.peek().Type == TokenLParen {
		p.next() // consume (
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().Type != TokenRParen {
			return nil, fmt.Errorf("expected ')', got %q", p.peek().Value)
		}
		p.next() // consume )
		return node, nil
	}

	if p.peek().Type == TokenWord {
		word := p.next()

		// Check if this is a field:op:value expression
		if p.peek().Type == TokenOp {
			op := p.next()
			if p.peek().Type != TokenWord || p.peek().Value == "" {
				return nil, fmt.Errorf("expected value after %s%s", word.Value, op.Value)
			}
			value := p.next()

			switch op.Value {
			case ":":
				// last:5m → relative time
				if strings.ToLower(word.Value) == "last" {
					dur, err := parseRelativeDuration(value.Value)
					if err != nil {
						return nil, fmt.Errorf("invalid duration %q: %v", value.Value, err)
					}
					return &Node{Type: NodeRelativeTime, Value: dur}, nil
				}
				return &Node{Type: NodeFieldMatch, Field: word.Value, Operator: ":", Value: value.Value}, nil
			case ">", ">=", "<", "<=":
				// timestamp>"..." → time comparison
				if timestampFields[strings.ToLower(word.Value)] {
					return &Node{Type: NodeTimeCompare, Field: word.Value, Operator: op.Value, Value: value.Value}, nil
				}
				return &Node{Type: NodeFieldCompare, Field: word.Value, Operator: op.Value, Value: value.Value}, nil
			case "~":
				return &Node{Type: NodeFieldRegex, Field: word.Value, Operator: "~", Value: value.Value}, nil
			default:
				return nil, fmt.Errorf("unknown operator: %q", op.Value)
			}
		}

		// Bare word = full-text search
		return &Node{Type: NodeFullText, Value: word.Value}, nil
	}

	return nil, fmt.Errorf("unexpected token: %q", p.peek().Value)
}

// parseRelativeDuration parses "5m", "1h", "30s", "2d" into a duration string
// that the evaluator will resolve against time.Now().
func parseRelativeDuration(s string) (string, error) {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("expected format like 5m, 1h, 30s, 2d")
	}
	n, _ := strconv.Atoi(m[1])
	if n <= 0 {
		return "", fmt.Errorf("duration must be positive")
	}
	return s, nil
}
