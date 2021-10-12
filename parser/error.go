// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"fmt"
	"strings"

	"github.com/golangee/dyml/token"
)

// UnexpectedTokenError is returned when a token appeared that the parser did not expect.
// It provides alternatives for tokens that were expected instead.
type UnexpectedTokenError struct {
	tok      token.Token
	expected []token.Type
}

// NewUnexpectedTokenError creates a new UnexpectedTokenError.
func NewUnexpectedTokenError(tok token.Token, expected ...token.Type) error {
	return UnexpectedTokenError{
		tok:      tok,
		expected: expected,
	}
}

func (u UnexpectedTokenError) Error() string {
	// Build a pretty string with expected tokens
	var expectedTokens []string

	for _, tt := range u.expected {
		tokenName := strings.TrimPrefix(string(tt), "Token")
		expectedTokens = append(expectedTokens, tokenName)
	}

	// Join the last two elements with an "or" to have a nice looking string.
	if len(expectedTokens) >= 2 {
		joined := fmt.Sprintf("%s or %s",
			expectedTokens[len(expectedTokens)-2],
			expectedTokens[len(expectedTokens)-1],
		)
		expectedTokens = expectedTokens[:len(expectedTokens)-1]
		expectedTokens[len(expectedTokens)-1] = joined
	}

	expected := strings.Join(expectedTokens, ", ")

	return fmt.Sprintf(
		"unexpected %s, expected %s",
		strings.TrimPrefix(string(u.tok.Type()), "Token"),
		expected)
}

// ForwardAttrError is returned when the token is a simple '@' for defining an attribute,
// but a forward definition '@@' is required.
type ForwardAttrError struct{}

func (e ForwardAttrError) Error() string {
	return "expected a forward attribute"
}

// NewForwardAttrError creates a new ForwardAttrError.
func NewForwardAttrError() error {
	return ForwardAttrError{}
}
