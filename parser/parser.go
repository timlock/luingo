package parser

import (
	"errors"
	"fmt"
	"io"
	"luingo/lexer"
	"luingo/vm"
	"math"
)

type Parser struct {
	lexer        lexer.Lexer
	constants    *constantTable
	byteCodes    []vm.ByteCode
	locals       []string
	localsIndex  map[string]byte
	stackPointer byte
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

func (p *Parser) loadExpTop(expression expression) error {
	return p.loadExp(p.stackPointer, expression)
}

func (p *Parser) loadConstExp(expression expression) (byte, bool) {
	switch expression.expressionType {
	case expressioinBoolean:
		return p.constants.addBoolean(expression.inner.(bool)), true
	case expressionFloat:
		return p.constants.addFloat(expression.inner.(float64)), true
	case expressionInteger:
		return p.constants.addInt(expression.inner.(int64)), true
	case expressionNil:
		return p.constants.addNil(), true
	case expressionString:
		return p.constants.addString(expression.inner.(string)), true
	default:
		return 0, false
	}
}

func (p *Parser) loadExp(destination byte, expression expression) error {
	switch expression.expressionType {
	case expressionNil:
		p.byteCodes = append(p.byteCodes, vm.LoadNil(destination))
	case expressioinBoolean:
		value := expression.inner.(bool)
		p.byteCodes = append(p.byteCodes, vm.LoadBool(destination, value))
	case expressionInteger:
		value := expression.inner.(int64)
		if value >= math.MinInt16 && value <= math.MaxInt16 {
			p.byteCodes = append(p.byteCodes, vm.LoadInt(destination, int16(value)))
		} else {
			p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addInt(value)))
		}
	case expressionFloat:
		value := expression.inner.(float64)
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addFloat(value)))
	case expressionString:
		value := expression.inner.(string)
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addString(value)))
	case expressionLocal:
		value := expression.inner.(byte)
		if value != destination {
			p.byteCodes = append(p.byteCodes, vm.Move(destination, value))
		}
	case expressionGlobal:
		value := expression.inner.(byte)
		p.byteCodes = append(p.byteCodes, vm.GetGlobal(destination, value))
	default:
		return fmt.Errorf("unknown expression type '%v'", expression.expressionType)
	}

	p.stackPointer = destination + 1
	return nil
}

func (p *Parser) readExpression() (expression, error) {
	token, err := p.lexer.Next()
	if err != nil {
		return expression{}, fmt.Errorf("reading function parameter: %w", err)
	}
	switch token.Type {
	case lexer.Nil:
		return newNilExpression(), nil
	case lexer.True:
		return newBooleanExpression(true), nil
	case lexer.False:
		return newBooleanExpression(false), nil
	case lexer.Integer:
		return newIntegerExpression(token.Integer), nil
	case lexer.Float:
		return newFloatExpression(token.Float), nil
	case lexer.String:
		return newStringExpression(token.Str), nil
	case lexer.Identifier:
		if pos, ok := p.localsIndex[token.Str]; ok {
			return newLocalExpression(pos), nil
		}
		return newGlobalExpression(p.constants.addString(token.Str)), nil
	case lexer.OpenBrace:
		tableExpr, err := p.tableConstructor()
		if err != nil {
			return expression{}, fmt.Errorf("loading table constructor: %w", err)
		}
		return tableExpr, nil

	default:
		return expression{}, fmt.Errorf("did not expect token %v' as expression", token.Type.String())
	}
}

func (p *Parser) tableConstructor() (expression, error) {
	tableStackIndex := byte(p.stackPointer + 1)
	p.byteCodes = append(p.byteCodes, vm.NewTableByteCode(tableStackIndex, 0, 0))
	var listCount byte
	var tableCount byte

loop:
	for {
		stackPointer := p.stackPointer

		peeked, err := p.lexer.Peek()
		if err != nil {
			return expression{}, err
		}

		switch peeked.Type {
		case lexer.ClosedBrace:
			p.lexer.Next()
			break loop
		case lexer.OpenSquareBracket:
			p.lexer.Next()
			tableCount++

			keyExpression, err := p.readExpression()
			if err != nil {
				return expression{}, fmt.Errorf("reading key expression: %w", err)
			}

			if _, err := p.lexer.ExpectToken(lexer.ClosedSquareBracket); err != nil {
				return expression{}, err
			}
			if _, err := p.lexer.ExpectToken(lexer.Assign); err != nil {
				return expression{}, err
			}

			byteCode, byteCodeConst, keyConstIndex, err := func() (func(byte, byte, byte) vm.ByteCode, func(byte, byte, byte) vm.ByteCode, byte, error) {
				switch keyExpression.expressionType {
				case expressionNil:
					return nil, nil, 0, errors.New("key may not be nil")

				case expressionFloat:
					value := keyExpression.inner.(float64)
					if math.IsNaN(value) {
						return nil, nil, 0, errors.New("number key may not be NaN")
					}
				case expressionInteger:
					value := keyExpression.inner.(int)
					if value <= math.MaxUint8 && value >= 0 {
						return vm.SetInt, vm.SetIntConst, byte(value), nil
					}
				case expressionString:
					value := keyExpression.inner.(string)
					constIndex := p.constants.addString(value)
					return vm.SetField, vm.SetFieldConst, constIndex, nil

				case expressionLocal:
					value := keyExpression.inner.(byte)
					return vm.SetTable, vm.SetTableConst, value, nil
				}

				if err := p.loadExp(p.stackPointer, keyExpression); err != nil {
					return nil, nil, 0, err
				}
				return vm.SetTable, vm.SetTableConst, p.stackPointer, nil

			}()
			if err != nil {
				return expression{}, err
			}

			valueExpression, err := p.readExpression()
			if err != nil {
				return expression{}, fmt.Errorf("reading value expression: %w", err)
			}

			if valueConstIndex, ok := p.loadConstExp(valueExpression); ok {
				p.byteCodes = append(p.byteCodes, byteCodeConst(tableStackIndex, keyConstIndex, valueConstIndex))
			} else if err := p.loadExpTop(valueExpression); err != nil {
				value := valueExpression.inner.(byte)
				p.byteCodes = append(p.byteCodes, byteCode(tableStackIndex, keyConstIndex, value))
			}

		case lexer.Identifier:
			keyOrValue := peeked
			p.lexer.Next()

			peeked, err := p.lexer.Peek()
			if err != nil {
				return expression{}, err
			}

			if peeked.Type == lexer.Assign {
				tableCount++
				p.lexer.Next()
				keyConstIndex := p.constants.addString(keyOrValue.Str)

				valueExpression, err := p.readExpression()
				if err != nil {
					return expression{}, fmt.Errorf("reading value expression: %w", err)
				}

				if valueConstIndex, ok := p.loadConstExp(valueExpression); ok {
					p.byteCodes = append(p.byteCodes, vm.SetFieldConst(tableStackIndex, keyConstIndex, valueConstIndex))
				} else if err := p.loadExpTop(valueExpression); err != nil {
					value := valueExpression.inner.(byte)
					p.byteCodes = append(p.byteCodes, vm.SetField(tableStackIndex, keyConstIndex, value))
				}
			} else {
				listCount++

				var valueExpression expression
				if pos, ok := p.localsIndex[keyOrValue.Str]; ok {
					valueExpression = newLocalExpression(pos)
				} else {
					valueExpression = newGlobalExpression(p.constants.addString(keyOrValue.Str))
				}

				if err := p.loadExp(stackPointer, valueExpression); err != nil {
					return expression{}, err
				}

				if listCount % 50 == 0 {
					p.byteCodes = append(p.byteCodes, vm.SetList(tableStackIndex, 50))
					p.stackPointer = tableStackIndex + 1
				}
			}
		default:
			listCount++
			p.lexer.Next()

			valueExpression, err := p.readExpression()
			if err != nil {
				return expression{}, fmt.Errorf("reading list item expression: %w", err)
			}
			if err := p.loadExp(stackPointer, valueExpression); err != nil {
				return expression{}, fmt.Errorf("loading list item expression: %w", err)
			}

			if listCount % 50 == 0 {
				p.byteCodes = append(p.byteCodes, vm.SetList(tableStackIndex, 50))
				p.stackPointer = tableStackIndex + 1
			}
		}

		peeked, err = p.lexer.Peek()
		if err != nil {
			return expression{}, err
		}

		switch peeked.Type {
		case lexer.Comma:
			fallthrough
		case lexer.SemiColon:
			p.lexer.Next()
		case lexer.ClosedBrace:
		default:
			return expression{}, fmt.Errorf("expected comma, semicolon or closed square brace but got '%v'", peeked.Type)
		}
	}

	remainingListItems := listCount % 50
	if remainingListItems > 0 {
		p.byteCodes = append(p.byteCodes, vm.SetList(tableStackIndex, remainingListItems))
	}

	p.byteCodes[tableStackIndex] = vm.NewTableByteCode(tableStackIndex, listCount, tableCount)
	return newLocalExpression(tableStackIndex), nil
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
		if token.Integer >= math.MinInt16 && token.Integer <= math.MaxInt16 {
			p.byteCodes = append(p.byteCodes, vm.LoadInt(destination, int16(token.Integer)))
		} else {
			p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addInt(token.Integer)))
		}
	case lexer.Float:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addFloat(token.Float)))
	case lexer.String:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addString(token.Str)))
	case lexer.Identifier:
		p.loadVar(destination, token.Str)
	case lexer.OpenBrace:
		if err := p.loadTableConstructor(destination); err != nil {
			return fmt.Errorf("loading table constructor: %w", err)
		}

	default:
		return fmt.Errorf("did not expect token %v' as expression", token.Type.String())
	}

	return nil
}

func (p *Parser) loadTableConstructor(tableStackIndex byte) error {
	p.byteCodes = append(p.byteCodes, vm.NewTableByteCode(tableStackIndex, 0, 0))
	var listCount byte
	var tableCount byte

	currentStackIndex := tableStackIndex
loop:
	for {
		peeked, err := p.lexer.Peek()
		if err != nil {
			return err
		}

		switch peeked.Type {
		case lexer.ClosedBrace:
			p.lexer.Next()
			break loop
		case lexer.OpenSquareBracket:
			p.lexer.Next()
			tableCount++

			currentStackIndex++
			if err := p.loadExpression(currentStackIndex); err != nil {
				return fmt.Errorf("loading table key expression: %w", err)
			}
			if _, err := p.lexer.ExpectToken(lexer.ClosedSquareBracket); err != nil {
				return err
			}
			if _, err := p.lexer.ExpectToken(lexer.Assign); err != nil {
				return err
			}

			currentStackIndex++
			if err := p.loadExpression(currentStackIndex); err != nil {
				return fmt.Errorf("loading table value expression: %w", err)
			}

			p.byteCodes = append(p.byteCodes, vm.SetTable(tableStackIndex, currentStackIndex-1, currentStackIndex))
		case lexer.Identifier:
			keyOrValue := peeked
			p.lexer.Next()

			peeked, err := p.lexer.Peek()
			if err != nil {
				return err
			}

			switch peeked.Type {
			case lexer.Assign:
				tableCount++
				p.lexer.Next()

				currentStackIndex++
				keyConstIndex := p.constants.addString(keyOrValue.Str)
				p.byteCodes = append(p.byteCodes, vm.LoadConst(currentStackIndex, keyConstIndex))
				if _, err := p.lexer.ExpectToken(lexer.Assign); err != nil {
					return err
				}

				currentStackIndex++
				if err := p.loadExpression(currentStackIndex); err != nil {
					return fmt.Errorf("loading table value expression: %w", err)
				}

				p.byteCodes = append(p.byteCodes, vm.SetField(tableStackIndex, keyConstIndex, currentStackIndex))
			case lexer.ClosedBrace:
				fallthrough
			case lexer.Comma:
				fallthrough
			case lexer.SemiColon:
				listCount++
				p.lexer.Next()

				currentStackIndex++
				p.loadVar(tableStackIndex, keyOrValue.Str)

				listValuesOnStack := currentStackIndex - tableStackIndex
				if listValuesOnStack > 50 {
					//clear stack
					p.byteCodes = append(p.byteCodes, vm.SetList(tableStackIndex, listValuesOnStack))
					currentStackIndex = tableStackIndex
				}
			}
		default:
			listCount++
			p.lexer.Next()

			currentStackIndex++
			if err := p.loadExpression(currentStackIndex); err != nil {
				return fmt.Errorf("loading table list expression: %w", err)
			}

			listValuesOnStack := currentStackIndex - tableStackIndex
			if listValuesOnStack > 50 {
				//clear stack
				p.byteCodes = append(p.byteCodes, vm.SetList(tableStackIndex, listValuesOnStack))
				currentStackIndex = tableStackIndex
			}
		}

		peeked, err = p.lexer.Peek()
		if err != nil {
			return err
		}

		switch peeked.Type {
		case lexer.Comma:
			fallthrough
		case lexer.SemiColon:
			p.lexer.Next()
		case lexer.ClosedSquareBracket:
		default:
			return fmt.Errorf("expected comma, semicolon or closed square bracket but got '%v'", peeked.Type)
		}
	}

	p.byteCodes = append(p.byteCodes, vm.SetList(tableStackIndex, currentStackIndex-tableStackIndex))
	currentStackIndex = tableStackIndex

	p.byteCodes[tableStackIndex] = vm.NewTableByteCode(tableStackIndex, listCount, tableCount)
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

func (c *constantTable) addBoolean(value bool) byte {
	if value {
		return c.addTrue()
	}

	return c.addFalse()
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

type expressionType byte

const (
	expressionNil expressionType = iota
	expressioinBoolean
	expressionInteger
	expressionFloat
	expressionString
	expressionLocal
	expressionGlobal
)

type expression struct {
	expressionType expressionType
	inner          any
}

func (e expression) isConst() bool {
	switch e.expressionType {
	case expressioinBoolean:
		fallthrough
	case expressionFloat:
		fallthrough
	case expressionInteger:
		fallthrough
	case expressionNil:
		fallthrough
	case expressionString:
		return true
	}
	return false
}

func (e expression) getLocal() (byte, bool) {
	if e.expressionType != expressionLocal {
		return 0, false
	}

	return e.inner.(byte), true
}

func newNilExpression() expression {
	return expression{expressionNil, nil}
}

func newBooleanExpression(value bool) expression {
	return expression{expressioinBoolean, value}
}

func newIntegerExpression(value int64) expression {
	return expression{expressionInteger, value}
}

func newFloatExpression(value float64) expression {
	return expression{expressionFloat, value}
}

func newStringExpression(value string) expression {
	return expression{expressionString, value}
}

func newLocalExpression(value byte) expression {
	return expression{expressionLocal, value}
}

func newGlobalExpression(value byte) expression {
	return expression{expressionGlobal, value}
}
