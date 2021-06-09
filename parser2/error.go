package parser2

import (
	"fmt"
	"strings"
)

// UnexpectedTokenError is returned when a token appeared that the parser did not expect.
// It provides alternatives for tokens that were expected instead.
type UnexpectedTokenError struct {
	tok      Token
	expected []TokenType
}

func NewUnexpectedTokenError(tok Token, expected ...TokenType) error {
	return UnexpectedTokenError{
		tok:      tok,
		expected: expected,
	}
}

func (u UnexpectedTokenError) Error() string {
	// Build a pretty string with expected tokens
	var expectedStrings []string
	for _, tt := range u.expected {
		expectedStrings = append(expectedStrings, string(tt))
	}

	expected := strings.Join(expectedStrings, ", ")

	return fmt.Sprintf(
		"unexpected %s, expected %s",
		u.tok.TokenType(),
		expected)
}

// ForwardAttrError is returned when the token is a simple '@' for defining an attribute,
// but a forward definition '@@' is required.
type ForwardAttrError struct{}

func (e ForwardAttrError) Error() string {
	return fmt.Sprintf("expected a forward attribute")
}

func NewForwardAttrError() error {
	return ForwardAttrError{}
}
