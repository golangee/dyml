package streamxmlencoder

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/golangee/tadl/parser2"
)

type XMLEncoder struct {
	lexer  *parser2.Lexer
	tokens []parser2.Token

	tadlText string
	xmlText  string
	prefix   string
	postfix  string
}

func NewEncoderFromString(text string) XMLEncoder {
	return XMLEncoder{
		lexer:    parser2.NewLexer("default", bytes.NewBuffer([]byte(text))),
		tadlText: text,
	}
}

func NewEncoderFromNameAndString(name, text string) XMLEncoder {
	return XMLEncoder{
		lexer:    parser2.NewLexer(name, bytes.NewBuffer([]byte(text))),
		tadlText: text,
	}
}

func (x *XMLEncoder) EncodeToXML() (string, error) {
	//err := x.Tokenize()
	var err error
	/*if err != nil {
		return "", err
	}*/
	var output string
	var parser = parser2.NewParser("test", bytes.NewBuffer([]byte(x.tadlText)))
	var tree *parser2.TreeNode
	tree, err = parser.Parse()
	if err != nil {
		return "", err
	}

	output, err = encodeRek(*tree)
	if err != nil {
		return "", err
	}
	return output, nil

	/*for i, Token := range x.tokens {
		if i == 0 {

		}
	}
	/*for{
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
	}*/
	return output, nil
}

func (x *XMLEncoder) Tokenize() error {
	for {
		currentToken, err := x.getNextToken()

		if err != nil {
			return err
		}
		if currentToken == nil {
			break
		}

		x.tokens = append(x.tokens, currentToken)
	}
	return nil
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

/*func (x *XMLEncoder) TokenToXML(lastToken, currentToken, nextToken parser2.Token) (string, error) {
	switch currentToken.TokenType() {
	case parser2.TokenIdentifier:
		return x.encodeIdentifier(currentToken.Pos())
	case parser2.TokenBlockStart:
		//return ("<" + lastToken.value + ">"), nil
		return "<Identifier>", nil
	case parser2.TokenBlockEnd:
		//return ("</" + lastToken.value + ">"), nil
		return "</Identifier>", nil
	case parser2.TokenGroupStart:

	case parser2.TokenGroupEnd:
	case parser2.TokenGenericStart:
	case parser2.TokenGenericEnd:
	case parser2.TokenG2Preamble:
	case parser2.TokenDefineElement:
	case parser2.TokenDefineAttribute:
	case parser2.TokenAssign:
	case parser2.TokenComma:
	case parser2.TokenCharData:
	case parser2.TokenG1Comment:
	case parser2.TokenG2Comment:
		//return ("<!-- " + nextToken.value + "-->"), nil
		return "<!-- Comment -->", nil
	case parser2.TokenG1LineEnd:
	}
	fmt.Print(currentToken.TokenType())
	return "_", nil
}

func (x *XMLEncoder) encodeIdentifier(position *token.Position) (string, error) {
	fmt.Println(position)
	fmt.Println(position.BeginPos.Line)
	fmt.Println(position.BeginPos.Col)
	fmt.Println(position.BeginPos.Offset)
	fmt.Println(position.EndPos.Line)
	fmt.Println(position.EndPos.Col)
	fmt.Println(position.EndPos.Offset)
	fmt.Println(x.tadlText[position.BeginPos.Offset-2 : position.EndPos.Offset-1])
	return "Identifier", nil
}*/

func encodeRek(root parser2.TreeNode) (string, error) {
	if root.IsComment() {
		return "<!-- " + *root.Comment + " -->", nil
	} else if root.IsText() {
		return *root.Text, nil
	} else if root.IsNode() {
		var outString, postfix string

		if root.BlockType == parser2.BlockNormal || root.BlockType == parser2.BlockNone {
			outString += "<" + root.Name
			postfix = "</" + root.Name + ">"
		} else if root.BlockType == parser2.BlockGroup {
			outString += "<" + root.Name + ` _groupType="()"`
			postfix = "</" + root.Name + ">"
		} else if root.BlockType == parser2.BlockGeneric {
			outString += "<" + root.Name + ` _groupType="<>"`
			postfix = "</" + root.Name + ">"
		}

		if len(root.Attributes) != 0 {
			for key, val := range root.Attributes {
				outString += " " + key + `="` + val + `"`
			}
		}

		outString += ">"
		if root.Name == "title" {
			fmt.Printf("TITLE: %v", root.Children)
		}
		for _, child := range root.Children {

			fmt.Printf("root: %v, child: %v", root, child)
			var text string
			text, err := encodeRek(*child)
			if err != nil {
				return "", err
			}

			outString += text
		}

		return outString + postfix, nil
	} else {
		return "", errors.New("Token not identified, aborting encoding")
	}
}
