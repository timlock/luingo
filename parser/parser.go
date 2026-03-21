package parser

import (
	"errors"
	"fmt"
	"io"
	"luingo/lexer"
	"luingo/vm"
)

type Parser struct {
	lexer       lexer.Lexer
	constants   *constantTable
	byteCodes   []vm.ByteCode
	locals      []string
	localsIndex map[string]byte
}

func NewParser(input string) *Parser {
	return &Parser{
		lexer:       *lexer.NewLexer(input),
		constants:   newConstantTable(),
		localsIndex: map[string]byte{},
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
			peeked, err := p.lexer.Peek()
			if err != nil {
				return nil, nil, fmt.Errorf("unexpected end after parsing an identifier: %w", err)
			}
			switch peeked.Type {
			case lexer.Assign:
				err = p.assignment(token.Str)
				if err != nil {
					return nil, nil, fmt.Errorf("parsing assignment: %w", err)
				}
			default:
				err = p.functionCall(token.Str)
				if err != nil {
					return nil, nil, fmt.Errorf("parsing function call: %w", err)
				}
			}

		case lexer.Local:
			if err := p.local(); err != nil {
				return nil, nil, fmt.Errorf("parsing local statement: %w", err)
			}
		case lexer.LineComment:
			// ignore comment
		default:
			return nil, nil, fmt.Errorf("%v did not expect token '%v'", p.lexer.Cursor(), token.Type.String())
		}
	}

	constants, byteCodes := p.constants.constants, p.byteCodes
	p.constants, p.byteCodes = newConstantTable(), nil

	return constants, byteCodes, nil
}

func (p *Parser) assignment(identifier string) error {
	if _, err := p.lexer.ExpectToken(lexer.Assign); err != nil {
		return err
	}
	if localIndex, ok := p.localsIndex[identifier]; ok {
		if err := p.loadExpression(localIndex); err != nil {
			return fmt.Errorf("reading right hand value of assignment to local '%v': %w", identifier, err)
		}

		return nil
	}

	destination := p.constants.addString(identifier)

	token, err := p.lexer.Next()
	if err != nil {
		return fmt.Errorf("reading right hand value of assignment to global '%v': %w", identifier, err)
	}

	switch token.Type {
	case lexer.Nil:
		p.byteCodes = append(p.byteCodes, vm.SetGlobalConst(destination, p.constants.addNil()))
	case lexer.True:
		p.byteCodes = append(p.byteCodes, vm.SetGlobalConst(destination, p.constants.addTrue()))
	case lexer.False:
		p.byteCodes = append(p.byteCodes, vm.SetGlobalConst(destination, p.constants.addFalse()))
	case lexer.Integer:
		p.byteCodes = append(p.byteCodes, vm.SetGlobalConst(destination, p.constants.addInt(token.Integer)))
	case lexer.Float:
		p.byteCodes = append(p.byteCodes, vm.SetGlobalConst(destination, p.constants.addFloat(token.Float)))
	case lexer.String:
		p.byteCodes = append(p.byteCodes, vm.SetGlobalConst(destination, p.constants.addString(token.Str)))
	case lexer.Identifier:
		if localIndex, ok := p.localsIndex[token.Str]; ok {
			p.byteCodes = append(p.byteCodes, vm.SetGlobal(destination, localIndex))
		} else {
			p.byteCodes = append(p.byteCodes, vm.SetGlobalGlobal(destination, p.constants.addString(token.Str)))
		}
	default:
		return fmt.Errorf("did not expect token %v' as expression", token.Type.String())
	}

	return nil
}

func (p *Parser) functionCall(identifier string) error {
	funcPos := byte(len(p.locals))
	p.loadVar(funcPos, identifier)

	next, err := p.lexer.Next()
	if err != nil {
		return fmt.Errorf("reading function parameter: %w", err)
	}

	switch next.Type {
	case lexer.String:

		p.byteCodes = append(p.byteCodes, vm.LoadConst(funcPos+1, p.constants.addString(next.Str)))

	case lexer.OpenBracket:
		if err := p.loadExpression(funcPos + 1); err != nil {
			return fmt.Errorf("loading expression in function call: %w", err)
		}

		next, err = p.lexer.ExpectToken(lexer.ClosedBracket)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%v did not expect token '%v'", p.lexer.Cursor(), next.Type.String())
	}

	p.byteCodes = append(p.byteCodes, vm.Call(funcPos, 1))
	return nil
}

func (p *Parser) local() error {
	next, err := p.lexer.ExpectToken(lexer.Identifier)
	if err != nil {
		return err
	}

	if _, err = p.lexer.ExpectToken(lexer.Assign); err != nil {
		return err
	}

	if err := p.loadExpression(byte(len(p.locals))); err != nil {
		return fmt.Errorf("loading expression assigned to local %v: %w", next.Str, err)
	}

	p.locals = append(p.locals, next.Str)
	p.localsIndex[next.Str] = byte(len(p.locals) - 1)
	return nil
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
			p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addInt(token.Integer)))
		}
	case lexer.Float:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addFloat(token.Float)))
	case lexer.String:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addString(token.Str)))
	case lexer.Identifier:
		p.loadVar(destination, token.Str)
	default:
		return fmt.Errorf("did not expect token %v' as expression", token.Type.String())
	}

	return nil
}

func (p *Parser) loadVar(destination byte, identifier string) {
	if pos, ok := p.localsIndex[identifier]; ok {
		p.byteCodes = append(p.byteCodes, vm.Move(destination, pos))
		return
	}

	p.byteCodes = append(p.byteCodes, vm.GetGlobal(destination, p.constants.addString(identifier)))
}

type constantTable struct {
	constants        []vm.Value
	nilConstantPos   *byte
	trueConstantPos  *byte
	falseConstantPos *byte
	stringConstants  map[string]byte
	integerConstants map[int64]byte
	floatConstants   map[float64]byte
}

func newConstantTable() *constantTable {
	return &constantTable{
		stringConstants:  map[string]byte{},
		integerConstants: map[int64]byte{},
		floatConstants:   map[float64]byte{},
	}
}

func (c *constantTable) addString(value string) byte {
	pos, ok := c.stringConstants[value]
	if !ok {
		c.constants = append(c.constants, vm.NewString(value))
		pos = byte(len(c.constants) - 1)
		c.stringConstants[value] = pos
	}

	return pos
}

func (c *constantTable) addNil() byte {
	if c.nilConstantPos != nil {
		return *c.nilConstantPos
	}
	c.constants = append(c.constants, vm.NewNil())
	pos := byte(len(c.constants) - 1)
	c.nilConstantPos = &pos
	return pos
}

func (c *constantTable) addTrue() byte {
	if c.trueConstantPos != nil {
		return *c.trueConstantPos
	}
	c.constants = append(c.constants, vm.NewBoolean(true))
	pos := byte(len(c.constants) - 1)
	c.trueConstantPos = &pos
	return pos
}

func (c *constantTable) addFalse() byte {
	if c.falseConstantPos != nil {
		return *c.falseConstantPos
	}
	c.constants = append(c.constants, vm.NewBoolean(false))
	pos := byte(len(c.constants) - 1)
	c.falseConstantPos = &pos
	return pos
}

func (c *constantTable) addInt(value int64) byte {
	pos, ok := c.integerConstants[value]
	if !ok {
		c.constants = append(c.constants, vm.NewInteger(value))
		pos = byte(len(c.constants) - 1)
		c.integerConstants[value] = pos
	}

	return pos
}

func (c *constantTable) addFloat(value float64) byte {
	pos, ok := c.floatConstants[value]
	if !ok {
		c.constants = append(c.constants, vm.NewFloat(value))
		pos = byte(len(c.constants) - 1)
		c.floatConstants[value] = pos
	}

	return pos
}