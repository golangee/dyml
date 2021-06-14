package streamxmlencoder

import (
	"bytes"
	"fmt"

	"github.com/golangee/tadl/parser2"
)

type XMLEncoder struct {
	lexer   *parser2.Lexer
	tokens  []parser2.Token
	xmlText string
}

func NewEncoderFromLexer(lexer parser2.Lexer) XMLEncoder {
	return XMLEncoder{
		lexer: &lexer,
	}
}

func NewEncoderFromString(text string) XMLEncoder {
	return XMLEncoder{
		lexer: parser2.NewLexer("default", bytes.NewBuffer([]byte(text))),
	}
}

func NewEncoderFromNameAndString(name, text string) XMLEncoder {
	return XMLEncoder{
		lexer: parser2.NewLexer(name, bytes.NewBuffer([]byte(text))),
	}
}

func (x *XMLEncoder) EncodeToXML() (string, error) {
	var output string

	for {
		currentToken, err := x.getNextToken()

		if err != nil {
			return "", err
		}
		if currentToken == nil {
			break
		}

		x.tokens = append(x.tokens, currentToken)
		XMLElement, err := x.TokenToXML(currentToken)
		if err != nil {
			return "", err
		}
		output += XMLElement
	}
	return output, nil
}

func (x *XMLEncoder) getNextToken() (parser2.Token, error) {
	token, err := x.lexer.Token()
	if err != nil {
		if token != nil {
			return nil, err
		}
		return nil, nil
	}
	fmt.Printf("token read: %v\n", token.TokenType())

	return token, nil
}

func (x *XMLEncoder) TokenToXML(currentToken parser2.Token) (string, error) {
	return "lulz", nil
}
