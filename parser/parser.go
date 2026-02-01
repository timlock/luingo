package parser

import (
	"errors"
	"fmt"
	"io"
	"luingo/lexer"
	"luingo/vm"
)

type Parser struct {
	lexer            lexer.Lexer
	constants        []vm.Value
	byteCodes        []vm.ByteCode
	stringConstants  map[string]byte
	integerConstants map[int64]byte
	floatConstants   map[float64]byte
	locals           []string
	localsIndex      map[string]byte
}

func NewParser(input string) *Parser {
	return &Parser{
		lexer:            *lexer.NewLexer(input),
		stringConstants:  map[string]byte{},
		integerConstants: map[int64]byte{},
		floatConstants:   map[float64]byte{},
		localsIndex:      map[string]byte{},
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
			funcPos := byte(len(p.locals))
			p.loadVar(funcPos, token.Str)

			next, err := p.lexer.Next()
			if err != nil {
				return nil, nil, fmt.Errorf("reading function parameter: %w", err)
			}

			switch next.Type {
			case lexer.String:

				p.byteCodes = append(p.byteCodes, vm.LoadConst(funcPos+1, p.addStringConstant(next.Str)))

			case lexer.OpenBracket:
				if err := p.loadExpression(funcPos + 1); err != nil {
					return nil, nil, fmt.Errorf("loading expression in function call: %w", err)
				}

				next, err = p.lexer.ExpectToken(lexer.ClosedBracket)
				if err != nil {
					return nil, nil, err
				}
			default:
				return nil, nil, fmt.Errorf("%v did not expect token '%v'", p.lexer.Cursor(), next.Type.String())
			}

			p.byteCodes = append(p.byteCodes, vm.Call(funcPos, 1))

		case lexer.Local:
			next, err := p.lexer.ExpectToken(lexer.Identifier)
			if err != nil {
				return nil, nil, err
			}

			if _, err = p.lexer.ExpectToken(lexer.Assign); err != nil {
				return nil, nil, err
			}

			if err := p.loadExpression(byte(len(p.locals))); err != nil {
				return nil, nil, fmt.Errorf("loading expression assigned to local %v: %w", next.Str, err)
			}

			p.locals = append(p.locals, next.Str)
			p.localsIndex[next.Str] = byte(len(p.locals) - 1)

		case lexer.LineComment:
			// ignore comment
		default:
			return nil, nil, fmt.Errorf("%v did not expect token '%v'", p.lexer.Cursor(), token.Type.String())
		}
	}

	constants, byteCodes := p.constants, p.byteCodes
	p.constants, p.byteCodes = nil, nil

	return constants, byteCodes, nil
}

func (p *Parser) loadExpression(destination byte) error {
	token, err := p.lexer.Next()
	if err != nil {
		return fmt.Errorf("reading function parameter: %w", err)
	}
	switch token.Type {
	case lexer.Nil:
		p.byteCodes = append(p.byteCodes, vm.LoadNil(destination))
	case lexer.True:
		p.byteCodes = append(p.byteCodes, vm.LoadBool(destination, true))
	case lexer.False:
		p.byteCodes = append(p.byteCodes, vm.LoadBool(destination, false))
	case lexer.Integer:
		if token.Integer >= -32768 && token.Integer <= 32767 {
			byteCode, err := vm.LoadInt(destination, int16(token.Integer))
			if err != nil {
				return err
			}
			p.byteCodes = append(p.byteCodes, byteCode)
		} else {
			p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.addIntegerConstant(token.Integer)))
		}
	case lexer.Float:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.addFloatConstant(token.Float)))
	case lexer.String:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.addStringConstant(token.Str)))
	case lexer.Identifier:
		p.loadVar(destination, token.Str)
	default:
		return fmt.Errorf("did not expect token %v' as expression", token.Type.String())
	}

	return nil
}

func (p *Parser) addStringConstant(value string) byte {
	pos, ok := p.stringConstants[value]
	if !ok {
		p.constants = append(p.constants, vm.NewString(value))
		pos = byte(len(p.constants) - 1)
		p.stringConstants[value] = pos
	}

	return pos
}

func (p *Parser) addIntegerConstant(value int64) byte {
	pos, ok := p.integerConstants[value]
	if !ok {
		p.constants = append(p.constants, vm.NewInteger(value))
		pos = byte(len(p.constants) - 1)
		p.integerConstants[value] = pos
	}

	return pos

}

func (p *Parser) addFloatConstant(value float64) byte {
	pos, ok := p.floatConstants[value]
	if !ok {
		p.constants = append(p.constants, vm.NewFloat(value))
		pos = byte(len(p.constants) - 1)
		p.floatConstants[value] = pos
	}

	return pos
}

func (p *Parser) loadVar(destination byte, identifier string) {
	if pos, ok := p.localsIndex[identifier]; ok {
		p.byteCodes = append(p.byteCodes, vm.Move(destination, pos))
		return
	}

	p.byteCodes = append(p.byteCodes, vm.GetGlobal(destination, p.addStringConstant(identifier)))
}
