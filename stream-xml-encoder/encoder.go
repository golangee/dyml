package streamxmlencoder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/golangee/tadl/parser2"
	"github.com/golangee/tadl/token"
)

// XMLEncoder translates tadl-input to corresponding XML
type XMLEncoder struct {
	lexer  *parser2.Lexer
	tokens []parser2.Token

	tadlText string
	xmlText  string
	prefix   string
	postfix  string

	identifiers []string
	postfixes   []string
	attributeBuffer []parser2.Token
}

// getIdentifier returns the last pushed Identifier
func (x *XMLEncoder) getFromStack(stack []string) (string, error) {
	if len(stack) == 0 {
		return "", nil
	}
	return stack[len(stack)-1], nil
}

// popIdentifier returns the last pushed Identifier and removes it
func (x *XMLEncoder) popFromStack(stack []string) (string, error) {
	if len(stack) == 0 {
		return "", nil
	}
	identifier := stack[len(stack)-1]
	stack = stack[:len(stack)-2]
	return identifier, nil
}

// pushIdentifier adds an Identifier to the stack
func (x *XMLEncoder) pushToStack(stack []string, i string) {
	stack = append(stack, i)
}

// NewEncoder creades a new XMLEncoder
// tadl-input is given as an io.Reader instance
func NewEncoder(filename string, r io.Reader) *XMLEncoder {
	buffer := new(strings.Builder)
	_, err := io.Copy(buffer, r)
	if err != nil {
		log.Fatal("Could not read from Reader. Aborting")
	}
	return &XMLEncoder{
		lexer:    parser2.NewLexer("default", r),
		tadlText: buffer.String(),
	}
}

// EncodeToXml uses a parser2.parser to create a syntax tree,
// utilizes the encodeRek method to translate it and returns the result
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

// encodeRek recursively translates the syntax tree
// given by its root Element to the corresponding XML.
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

// Tokenize creates a Slice of consecutive Tokens, representing the tadl-input syntax
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

/*
// GetNextTokenToXML returns the next XML Translation
// to the corresponding TADL token in the input stream.
func (x *XMLEncoder) GetNextTokenToXML() (string, error) {
	token, err := x.getNextToken()
	if err != nil {
		return "", err
	}

	xmlstring, err = x.TokenToXML(token)

}*/

// getNextToken uses a Lexer to read the next consecutive Token
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
	outString := ""
	switch currentToken.TokenType() {
	case parser2.TokenIdentifier:
		ct, _ := currentToken.(*parser2.Identifier)
		x.pushToStack(x.identifiers, ct.Value)
		outString = "<" + ct.Value

		nextToken, err := x.getNextToken()
		if err != nil {
			return "", err
		}

		switch nextToken.TokenType(){
			case parser2.TokenDefineAttribute:
				nextToken = nextToken.(*parser2.DefineAttribute)
				if !nextToken.Forward {
					if nextToken = x.getNextToken(); nextToken.TokenType() != parser2.TokenIdentifier{
						return "", errors.New("Unexpected Token, expected Identifier")
					}
					nextToken = nextToken.(*parser2.Identifier)
					outString += " " + nextToken.Value + "="

					if nextToken = x.getNextToken(); nextToken.TokenType() != parser2.TokenBlockStart{
						return "", errors.New("Unexpected Token, expected BlockStart")
					}

					if nextToken = x.getNextToken(); nextToken.TokenType() != parser2.TokenCharData{
						return "", errors.New("Unexpected Token, expected Chardata")
					}
					nextToken = nextToken.(*parser2.CharData)

					outString += '"' + nextToken.Value + `"`

				}else {
					if nextToken = x.getNextToken(); nextToken.TokenType() != parser2.TokenIdentifier{
						return "", errors.New("Unexpected Token, expected Identifier")
					}
					identifier = nextToken.(*parser2.Identifier).Value

					if nextToken = x.getNextToken(); nextToken.TokenType() != parser2.TokenBlockStart{
						return "", errors.New("Unexpected Token, expected BlockStart")
					}

					if nextToken = x.getNextToken(); nextToken.TokenType() != parser2.TokenCharData{
						return "", errors.New("Unexpected Token, expected Chardata")
					}
				

					x.pushToStack(attributeBuffer, nextToken.(*parser2.CharData).Value)
					x.pushToStack(x.attributeBuffer, identifier)
				}
		
			case parser2.TokenBlockStart:
				outString += ">"
			case parser2.TokenIdentifier:
				outString += "><" + ct.Value + "/>"
		}
		


	}




		return "", nil

	case parser2.TokenBlockStart:
		identifier, err := x.getFromStack(x.identifiers)
		if err != nil {
			return "", err
		}
		x.pushToStack(x.postfixes, ">")
		return ("<" + identifier), nil

	case parser2.TokenBlockEnd:
		identifier, err := x.popFromStack(x.identifiers)
		if err != nil {
			return "", err
		}
		return ("</" + identifier + ">"), nil

	case parser2.TokenGroupStart:
		return
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

/*func (x *XMLEncoder) encodeIdentifier(position *token.Position) (string, error) {
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