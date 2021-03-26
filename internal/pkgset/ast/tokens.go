package ast

import (
	"fmt"
	"strings"
)

type Token struct {
	Kind Kind
	Text string
}

type Kind byte

const (
	TUnknown    Kind = '?'
	TOp         Kind = 'o'
	TComma      Kind = ','
	TSelector   Kind = 's'
	TFunc       Kind = 'f'
	TLeftParen  Kind = '('
	TRightParen Kind = ')'
	TPackage    Kind = 'p'
)

func (k Kind) String() string { return string(k) }

func Tokenize(s string) ([]Token, error) {
	var tokens []Token
	emit := func(kind Kind, text string) {
		tokens = append(tokens, Token{kind, text})
	}

	p := 0
	for p < len(s) {
		// skip whitespace
		for p < len(s) && s[p] == ' ' {
			p++
		}
		// finish when everything is parsed
		if p >= len(s) {
			break
		}

		var ident string
		p, ident = parseIdent(p, s)
		if ident != "" {
			if p < len(s) && s[p] == '(' {
				emit(TFunc, ident)
				continue
			}
			if strings.Contains(ident, "=") {
				return tokens, fmt.Errorf("package name %q shouldn't contain '='", ident)
			}
			emit(TPackage, ident)
			continue
		}

		switch s[p] {
		case '(':
			p++
			emit(TLeftParen, "(")
		case ')':
			p++
			emit(TRightParen, ")")
		case ':':
			p++
			var selector string
			p, selector = parseSelector(p, s)
			if selector == "" {
				return tokens, fmt.Errorf("expected selector %d", p)
			}
			emit(TSelector, selector)
		case '+', '-':
			op := string(s[p])
			p++
			if p < len(s) && s[p] == '(' {
				emit(TFunc, op)
				continue
			}
			emit(TOp, op)
		case ',':
			p++
			emit(TComma, ",")
		default:
			return tokens, fmt.Errorf("unknown symbol at %d: %s", p, string(s[p]))
		}
	}

	return tokens, nil
}

func isIdentFirst(p byte) bool {
	return (p == '.') ||
		('a' <= p && p <= 'z') || ('A' <= p && p <= 'Z') || ('0' <= p && p <= '9')
}

func isPrefixOp(p byte) bool {
	return p == '+' || p == '-'
}

func isIdent(p byte) bool {
	return (p == '.') || (p == '@') || (p == '_') || (p == '-') || (p == '/') ||
		('a' <= p && p <= 'z') || ('A' <= p && p <= 'Z') || ('0' <= p && p <= '9') ||
		(p == '=') // for build tags
}

func parseIdent(start int, s string) (int, string) {
	if start >= len(s) {
		return start, ""
	}

	if !isIdentFirst(s[start]) {
		return start, ""
	}

	p := start
	for p < len(s) && isIdent(s[p]) {
		p++
	}
	return p, s[start:p]
}

func parseSelector(start int, s string) (int, string) {
	if start >= len(s) {
		return start, ""
	}

	if !isIdentFirst(s[start]) && !isPrefixOp(s[start]) {
		return start, ""
	}

	p := start + 1
	for p < len(s) && isIdent(s[p]) {
		p++
	}
	return p, s[start:p]
}
