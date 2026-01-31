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
	stringConstants  map[string]struct{}
	integerConstants map[int64]struct{}
	floatConstants   map[float64]struct{}
	locals           []string
}

func NewParser(input string) *Parser {
	return &Parser{
		lexer:            *lexer.NewLexer(input),
		stringConstants:  map[string]struct{}{},
		integerConstants: map[int64]struct{}{},
		floatConstants:   map[float64]struct{}{},
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
			p.loadString(token.Str)
			p.byteCodes = append(p.byteCodes, vm.GetGlobal(0, byte(len(p.byteCodes) -1)))

			next, err := p.lexer.Next()
			if err != nil {
				return nil, nil, fmt.Errorf("reading function parameter: %w", err)
			}

			switch next.Type {
			case lexer.String:
				p.loadString(next.Str)
				p.byteCodes = append(p.byteCodes, vm.Call(0, 1))

			case lexer.OpenBracket:
				if err := p.loadExpression(); err != nil {
					return nil, nil, fmt.Errorf("loading expression in function call: %w", err)
				}

				next, err = p.lexer.ExpectToken(lexer.ClosedBracket)
				if err != nil {
					return nil, nil, err
				}

				p.byteCodes = append(p.byteCodes, vm.Call(0, 1))

			case lexer.Local:
				next, err = p.lexer.ExpectToken(lexer.Assign)
				if err != nil {
					return nil, nil, err
				}

				if err := p.loadExpression(); err != nil {
					return nil, nil, fmt.Errorf("loading expression assigned to local %v: %w", next.Str, err)
				}

				p.locals = append(p.locals, next.Str)

			default:
				return nil, nil, fmt.Errorf("did not expect token %v' after identifier", next.Type.String())
			}

		}
	}

	constants, byteCodes := p.constants, p.byteCodes
	p.constants, p.byteCodes = nil, nil

	return constants, byteCodes, nil
}

func (p *Parser) loadExpression() error {
	token, err := p.lexer.Next()
	if err != nil {
		return fmt.Errorf("reading function parameter: %w", err)
	}
	switch token.Type {
	case lexer.Nil:
		p.byteCodes = append(p.byteCodes, vm.LoadNil(1))
	case lexer.True:
		p.byteCodes = append(p.byteCodes, vm.LoadBool(1, true))
	case lexer.False:
		p.byteCodes = append(p.byteCodes, vm.LoadBool(1, false))
	case lexer.Integer:
		if token.Integer >= 0 && token.Integer <= 65535 {
			byteCode, err := vm.LoadUInt(1, uint16(token.Integer))
			if err != nil {
				return err
			}
			p.byteCodes = append(p.byteCodes, byteCode)
		} else if token.Integer >= -32768 && token.Integer <= 32767 {
			byteCode, err := vm.LoadInt(1, int16(token.Integer))
			if err != nil {
				return err
			}
			p.byteCodes = append(p.byteCodes, byteCode)
		} else {
			p.loadInteger(token.Integer)
		}
	case lexer.Float:
		p.loadFloat(token.Float)
	case lexer.String:
		p.loadString(token.Str)
	case lexer.Identifier:
		//TODO
	default:
		return fmt.Errorf("did not expect token %v' as expression", token.Type.String())
	}

	return nil
}

func (p *Parser) loadString(value string) {
	if _, ok := p.stringConstants[value]; !ok {
		p.stringConstants[value] = struct{}{}

		p.loadConst(vm.NewString(value))
	}
}

func (p *Parser) loadInteger(value int64) {
	if _, ok := p.integerConstants[value]; !ok {
		p.integerConstants[value] = struct{}{}

		p.loadConst(vm.NewInteger(value))
	}
}

func (p *Parser) loadFloat(value float64) {
	if _, ok := p.floatConstants[value]; !ok {
		p.floatConstants[value] = struct{}{}

		p.loadConst(vm.NewFloat(value))
	}
}

func (p *Parser) loadConst(value vm.Value) {
	p.constants = append(p.constants, value)
	p.byteCodes = append(p.byteCodes, vm.LoadConst(1, byte(len(p.constants)-1)))
}
