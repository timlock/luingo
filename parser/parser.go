package parser

import (
	"errors"
	"fmt"
	"io"
	"luingo/lexer"
	"luingo/vm"
)

type Parser struct {
	lexer     lexer.Lexer
	constants []vm.Value
	byteCodes []vm.ByteCode
}

func NewParser(input string) *Parser {
	return &Parser{
		lexer: *lexer.NewLexer(input),
	}
}

func (p *Parser) Parse() ([]vm.Value, []vm.ByteCode, error) {
	for {
		token, err := p.lexer.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, nil, fmt.Errorf("reading next token: %w", err)
		}

		switch token.Type {
		case lexer.Identifier:
			p.constants = append(p.constants, vm.String(token.Str))
			p.byteCodes = append(p.byteCodes, vm.GetGlobal(0, byte(len(p.constants)-1)))

			next, err := p.lexer.ExpectToken(lexer.String)
			if err != nil {
				return nil, nil, fmt.Errorf("reading string token: %w", err)
			}

			p.constants = append(p.constants, vm.String(next.Str))
			p.byteCodes = append(p.byteCodes, vm.LoadConst(1, byte(len(p.constants)-1)))
			p.byteCodes = append(p.byteCodes, vm.Call(0, 1))

		}
	}

	constants, byteCodes := p.constants, p.byteCodes
	p.constants, p.byteCodes = nil, nil

	return constants, byteCodes, nil
}
