package parser

import (
	"errors"
	"fmt"
	"io"
	"luingo/lexer"
	"luingo/vm"
	"math"
)

type Error struct {
	inner  error
	cursor lexer.Cursor
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v %+v", e.inner, e.cursor)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.inner
}

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
		case lexer.SemiColon:

		case lexer.OpenBracket:
			fallthrough
		case lexer.Identifier:

			prefixExp, err := p.prefixExp(token)
			if err != nil {
				return nil, nil, fmt.Errorf("parsing prefixexp: %w", err)
			}
			if prefixExp.expressionType != expressionCall {
				p.assignment(prefixExp)
			}

		case lexer.Local:
			if err := p.local(); err != nil {
				return nil, nil, fmt.Errorf("parsing local statement: %w", err)
			}
		default:
			return nil, nil, p.newError(fmt.Errorf("did not expect token '%v'", token.Type.String()))
		}

		p.stackPointer = byte(len(p.locals))
	}

	constants, byteCodes := p.constants.constants, p.byteCodes
	p.constants, p.byteCodes = newConstantTable(), nil

	return constants, byteCodes, nil
}

func (p *Parser) assignment(firstVariable expression) error {
	varList := []expression{firstVariable}
loop:
	for {
		token, err := p.lexer.Next()
		if err != nil {
			return err
		}

		switch token.Type {
		case lexer.Comma:
			token, err := p.lexer.Next()
			if err != nil {
				return err
			}
			expression, err := p.prefixExp(token)
			if err != nil {
				return err
			}
			varList = append(varList, expression)

		case lexer.Assign:
			break loop

		default:
			return fmt.Errorf("unexpected token in varlist '%v'", token.Type)
		}
	}

	stackPointer := p.stackPointer
	var expListSize byte = 0
	var lastExpression expression

	for {
		var err error
		lastExpression, err = p.readExpression()
		if err != nil {
			return err
		}

		peeked, err := p.lexer.Peek()
		if err != nil {
			return err
		}

		if peeked.Type != lexer.Comma {
			break
		}

		p.lexer.Next()
		p.loadExpression(stackPointer+expListSize, lastExpression)
		expListSize++
	}

	if expListSize+1 == byte(len(varList)) {
		lastVar := varList[len(varList)-1]
		varList = varList[:len(varList)-1]
		if err := p.assignVariable(lastVar, lastExpression); err != nil {
			return err
		}
	} else if expListSize+1 > byte(len(varList)) {
		expListSize = byte(len(varList))
	} else {
		nilExpressions := byte(len(varList)) - expListSize + 1
		for i := range nilExpressions {
			p.loadExpression(stackPointer+expListSize+1+i, newNilExpression())
		}
	}

	for len(varList) > 0 {
		var (
			lastVar expression
			ok      bool
		)
		varList, lastVar, ok = pop(varList)
		if !ok {
			break
		}

		expListSize--
		p.assignVariableLocal(lastVar, stackPointer+expListSize)
	}

	return nil
}

func pop[T any](s []T) ([]T, T, bool) {
	if len(s) == 0 {
		var empty T
		return nil, empty, false
	}

	last := s[len(s)-1]
	s = s[:len(s)-1]
	return s, last, true
}

func (p *Parser) assignVariable(variable, value expression) error {
	switch variable.expressionType {
	case expressionLocal:
		p.loadExpression(variable.inner.(byte), value)
	default:
		index, isConst, err := p.addConstOrLoadExp(value)
		if err != nil {
			return err
		}

		if isConst {
			return p.assignVariableConst(variable, index)
		}
		return p.assignVariableLocal(variable, index)
	}

	return nil
}

func (p *Parser) assignVariableLocal(variable expression, stackIndex byte) error {
	switch variable.expressionType {
	case expressionLocal:
		p.byteCodes = append(p.byteCodes, vm.Move(variable.inner.(byte), stackIndex))

	case expressionGlobal:
		p.byteCodes = append(p.byteCodes, vm.SetGlobal(variable.inner.(byte), stackIndex))

	case expressionIndex:
		pair := variable.inner.([2]byte)
		p.byteCodes = append(p.byteCodes, vm.SetTable(pair[0], pair[1], stackIndex))

	case expressionIndexField:
		pair := variable.inner.([2]byte)
		p.byteCodes = append(p.byteCodes, vm.SetField(pair[0], pair[1], stackIndex))

	case expressionIndexInt:
		pair := variable.inner.([2]byte)
		p.byteCodes = append(p.byteCodes, vm.SetInt(pair[0], pair[1], stackIndex))

	default:
		return fmt.Errorf("did not expect expression '%v' in assignment to local variable", variable.expressionType)
	}

	return nil
}

func (p *Parser) assignVariableConst(variable expression, constIndex byte) error {
	switch variable.expressionType {
	case expressionGlobal:
		p.byteCodes = append(p.byteCodes, vm.SetGlobalConst(variable.inner.(byte), constIndex))

	case expressionIndex:
		pair := variable.inner.([2]byte)
		p.byteCodes = append(p.byteCodes, vm.SetTableConst(pair[0], pair[1], constIndex))

	case expressionIndexField:
		pair := variable.inner.([2]byte)
		p.byteCodes = append(p.byteCodes, vm.SetFieldConst(pair[0], pair[1], constIndex))

	case expressionIndexInt:
		pair := variable.inner.([2]byte)
		p.byteCodes = append(p.byteCodes, vm.SetIntConst(pair[0], pair[1], constIndex))

	default:
		return fmt.Errorf("did not expect expression '%v' in assignment to const variable", variable.expressionType)
	}

	return nil
}

func (p *Parser) local() error {
	var variables []string
	var valuesSize byte
loop:
	for {
		token, err := p.lexer.ExpectToken(lexer.Identifier)
		if err != nil {
			return err
		}
		variables = append(variables, token.Str)

		peeked, err := p.lexer.Peek()
		if err != nil {
			return err
		}

		switch peeked.Type {
		case lexer.Comma:
			p.lexer.Next()

		case lexer.Assign:
			p.lexer.Next()
			valuesSize, err = p.expList()
			if err != nil {
				return err
			}
			break loop
		default:
			break loop
		}
	}

	if valuesSize < byte(len(variables)) {
		nilsSize := byte(len(variables)) - valuesSize
		for i := 0; i < int(nilsSize); i++ {
			stackIndex := byte(len(p.locals)) + valuesSize + byte(i)
			p.byteCodes = append(p.byteCodes, vm.LoadNil(stackIndex))
		}
	}

	for _, local := range variables {
		p.locals = append(p.locals, local)
		p.localsIndex[local] = byte(len(p.locals) - 1)
	}

	return nil
}

func (p *Parser) loadExpTop(expression expression) (byte, error) {
	return p.loadExpIfNotLocal(p.stackPointer, expression)
}

func (p *Parser) loadExpIfNotLocal(destination byte, expression expression) (byte, error) {
	if expression.expressionType == expressionLocal {
		return expression.inner.(byte), nil
	}

	p.loadExpression(destination, expression)

	return destination, nil
}

func (p *Parser) addConstOrLoadExp(expression expression) (byte, bool, error) {
	switch expression.expressionType {
	case expressioinBoolean:
		return p.constants.addBoolean(expression.inner.(bool)), true, nil
	case expressionFloat:
		return p.constants.addFloat(expression.inner.(float64)), true, nil
	case expressionInteger:
		return p.constants.addInt(expression.inner.(int64)), true, nil
	case expressionNil:
		return p.constants.addNil(), true, nil
	case expressionString:
		return p.constants.addString(expression.inner.(string)), true, nil
	default:
		stackIndex, err := p.loadExpTop(expression)
		if err != nil {
			return 0, false, err
		}
		return stackIndex, false, nil
	}
}

func (p *Parser) prefixExp(token lexer.Token) (expression, error) {
	stackPointer := p.stackPointer

	var exp expression
	switch token.Type {
	case lexer.Identifier:
		if pos, ok := p.localsIndex[token.Str]; ok {
			exp = newLocalExpression(pos)
		} else {
			exp = newGlobalExpression(p.constants.addString(token.Str))
		}

	case lexer.OpenBracket:
		var err error
		exp, err = p.readExpression()
		if err != nil {
			return expression{}, err
		}
		if _, err := p.lexer.ExpectToken(lexer.ClosedBracket); err != nil {
			return expression{}, err
		}

	default:
		return expression{}, p.newError(fmt.Errorf("did not expect '%v' in prefixexp", token.Type))
	}

	for {
		peeked, err := p.lexer.Peek()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return exp, nil
			}
			return expression{}, err
		}

		switch peeked.Type {
		case lexer.OpenSquareBracket:
			p.lexer.Next()

			tableStackIndex, err := p.loadExpIfNotLocal(stackPointer, exp)
			if err != nil {
				return expression{}, err
			}
			exp, err = p.readExpression()
			if err != nil {
				return expression{}, err
			}

			if exp.expressionType == expressionString {
				exp = newIndexFieldExpression(tableStackIndex, p.constants.addString(exp.inner.(string)))
			} else if exp.expressionType == expressionInteger && exp.inner.(int64) <= math.MaxUint8 && exp.inner.(int64) >= 0 {
				exp = newIndexIntExpression(tableStackIndex, byte(exp.inner.(int64)))
			} else {
				stackIndex, err := p.loadExpTop(exp)
				if err != nil {
					return expression{}, err
				}
				exp = newIndexExpression(tableStackIndex, stackIndex)
			}

			if _, err := p.lexer.ExpectToken(lexer.ClosedSquareBracket); err != nil {
				return expression{}, err
			}

		case lexer.Dot:
			p.lexer.Next()

			identifierToken, err := p.lexer.ExpectToken(lexer.Identifier)
			if err != nil {
				return expression{}, err
			}
			tableStackIndex, err := p.loadExpIfNotLocal(stackPointer, exp)
			if err != nil {
				return expression{}, err
			}
			exp = newIndexFieldExpression(tableStackIndex, p.constants.addString(identifierToken.Str))

		case lexer.OpenBracket:
			fallthrough
		case lexer.OpenBrace:
			fallthrough
		case lexer.String:
			p.loadExpression(stackPointer, exp)
			exp, err = p.args()
			if err != nil {
				return expression{}, err
			}

		default:
			return exp, nil
		}
	}
}

func (p *Parser) args() (expression, error) {
	var argCount byte
	funcStackIndex := p.stackPointer - 1

	token, err := p.lexer.Next()
	if err != nil {
		return expression{}, err
	}

	switch token.Type {
	case lexer.OpenBracket:
		peeked, err := p.lexer.Peek()
		if err != nil {
			return expression{}, err
		}
		if peeked.Type == lexer.ClosedBracket {
			p.lexer.Next()
		} else {
			argCount, err = p.expList()
			if err != nil {
				return expression{}, err
			}

			if _, err := p.lexer.ExpectToken(lexer.ClosedBracket); err != nil {
				return expression{}, err
			}
		}
	case lexer.OpenBrace:
		p.tableConstructor()
		argCount = 1
	case lexer.String:
		p.loadExpression(funcStackIndex+1, newStringExpression(token.Str))
		argCount = 1
	default:
		return expression{}, p.newError(fmt.Errorf("invalid args token '%v'", token.Type))
	}

	p.byteCodes = append(p.byteCodes, vm.Call(funcStackIndex, argCount))

	return newCallExpression(), nil
}

func (p *Parser) expList() (byte, error) {
	stackPointer := p.stackPointer

	var size byte
	for {
		exp, err := p.readExpression()
		if err != nil {
			return 0, err
		}
		p.loadExpression(stackPointer+size, exp)
		size++

		peeked, err := p.lexer.Peek()
		if err != nil {
			return 0, err
		}

		if peeked.Type != lexer.Comma {
			return size, nil
		}

		p.lexer.Next()
	}
}

func (p *Parser) loadExpression(destination byte, expression expression) {
	switch expression.expressionType {
	case expressionNil:
		p.byteCodes = append(p.byteCodes, vm.LoadNil(destination))

	case expressioinBoolean:
		p.byteCodes = append(p.byteCodes, vm.LoadBool(destination, expression.inner.(bool)))

	case expressionInteger:
		value := expression.inner.(int64)
		if value >= math.MinInt16 && value <= math.MaxInt16 {
			p.byteCodes = append(p.byteCodes, vm.LoadInt(destination, int16(value)))
		} else {
			p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addInt(value)))
		}

	case expressionFloat:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addFloat(expression.inner.(float64))))

	case expressionString:
		p.byteCodes = append(p.byteCodes, vm.LoadConst(destination, p.constants.addString(expression.inner.(string))))

	case expressionLocal:
		value := expression.inner.(byte)
		if value != destination {
			p.byteCodes = append(p.byteCodes, vm.Move(destination, value))
		}

	case expressionGlobal:
		p.byteCodes = append(p.byteCodes, vm.GetGlobal(destination, expression.inner.(byte)))

	case expressionCall:

	case expressionIndex:
		pair := expression.inner.([2]byte)
		tableStackIndex := pair[0]
		keyStackIndex := pair[1]

		p.byteCodes = append(p.byteCodes, vm.GetTable(destination, tableStackIndex, keyStackIndex))

	case expressionIndexField:
		pair := expression.inner.([2]byte)
		tableStackIndex := pair[0]
		keyConstIndex := pair[1]

		p.byteCodes = append(p.byteCodes, vm.GetField(destination, tableStackIndex, keyConstIndex))

	case expressionIndexInt:
		pair := expression.inner.([2]byte)
		tableStackIndex := pair[0]
		integer := pair[1]

		p.byteCodes = append(p.byteCodes, vm.GetInt(destination, tableStackIndex, integer))

	default:
		panic(fmt.Sprintf("unexpected parser.expressionType: %#v", expression.expressionType))
	}

	p.stackPointer = destination + 1
}

func (p *Parser) readExpression() (expression, error) {
	token, err := p.lexer.Next()
	if err != nil {
		return expression{}, fmt.Errorf("reading function parameter: %w", err)
	}
	return p.expression(token)
}

func (p *Parser) expression(token lexer.Token) (expression, error) {
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
	case lexer.OpenBrace:
		tableExpr, err := p.tableConstructor()
		if err != nil {
			return expression{}, fmt.Errorf("loading table constructor: %w", err)
		}
		return tableExpr, nil

	default:
		prefixExpression, err := p.prefixExp(token)
		if err != nil {
			return expression{}, err
		}

		return prefixExpression, nil
	}
}

func (p *Parser) tableConstructor() (expression, error) {
	tableStackIndex := p.stackPointer
	p.stackPointer++
	p.byteCodes = append(p.byteCodes, vm.NewTableByteCode(tableStackIndex, 0, 0))
	newTableByteCodeIndex := len(p.byteCodes) - 1

	var listCount, tableCount byte
loop:
	for {
		stackPointer := p.stackPointer

		peeked, err := p.lexer.Peek()
		if err != nil {
			return expression{}, err
		}

		var (
			keyOrValueExpression expression
			isKey                bool
		)

		switch peeked.Type {
		case lexer.ClosedBrace:
			p.lexer.Next()
			break loop
		case lexer.OpenSquareBracket:
			p.lexer.Next()

			keyOrValueExpression, err = p.readExpression()
			if err != nil {
				return expression{}, fmt.Errorf("reading key expression: %w", err)
			}
			if _, err := p.lexer.ExpectToken(lexer.ClosedSquareBracket); err != nil {
				return expression{}, err
			}
			if _, err := p.lexer.ExpectToken(lexer.Assign); err != nil {
				return expression{}, err
			}
			isKey = true

		case lexer.Identifier:
			keyOrValue := peeked
			p.lexer.Next()

			peeked, err := p.lexer.Peek()
			if err != nil {
				return expression{}, err
			}

			if peeked.Type == lexer.Assign {
				p.lexer.Next()
				keyOrValueExpression = newStringExpression(keyOrValue.Str)
				isKey = true
			} else {
				keyOrValueExpression, err = p.expression(peeked)
				if err != nil {
					return expression{}, fmt.Errorf("reading list item expression: %w", err)
				}
			}
		default:
			keyOrValueExpression, err = p.readExpression()
			if err != nil {
				return expression{}, fmt.Errorf("reading list item expression: %w", err)
			}
		}

		if isKey {
			tableCount++

			byteCode, byteCodeConst, keyPart, err := func() (func(byte, byte, byte) vm.ByteCode, func(byte, byte, byte) vm.ByteCode, byte, error) {
				switch keyOrValueExpression.expressionType {
				case expressionNil:
					return nil, nil, 0, errors.New("key may not be nil")
				case expressionFloat:
					if math.IsNaN(keyOrValueExpression.inner.(float64)) {
						return nil, nil, 0, errors.New("number key may not be NaN")
					}
				case expressionString:
					return vm.SetField, vm.SetFieldConst, p.constants.addString(keyOrValueExpression.inner.(string)), nil
				case expressionLocal:
					return vm.SetTable, vm.SetTableConst, keyOrValueExpression.inner.(byte), nil
				case expressionInteger:
					intValue := keyOrValueExpression.inner.(int)
					if intValue <= math.MaxUint8 && intValue >= 0 {
						return vm.SetInt, vm.SetIntConst, byte(intValue), nil
					}
				}
				p.loadExpression(p.stackPointer, keyOrValueExpression)
				return vm.SetTable, vm.SetTableConst, p.stackPointer, nil
			}()
			if err != nil {
				return expression{}, err
			}

			valueExpression, err := p.readExpression()
			if err != nil {
				return expression{}, fmt.Errorf("reading value expression: %w", err)
			}

			valuePart, valueIsConst, err := p.addConstOrLoadExp(valueExpression)
			if err != nil {
				return expression{}, err
			}
			if valueIsConst {
				p.byteCodes = append(p.byteCodes, byteCodeConst(tableStackIndex, keyPart, valuePart))
			} else {
				p.byteCodes = append(p.byteCodes, byteCode(tableStackIndex, keyPart, valuePart))
			}

		} else {
			listCount++

			p.loadExpression(stackPointer, keyOrValueExpression)

			if listCount%50 == 0 {
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
			p.lexer.Next()
			break loop
		default:
			return expression{}, p.newError(fmt.Errorf("expected comma, semicolon or closed square brace but got '%v'", peeked.Type))
		}
	}

	remainingListItems := listCount % 50
	if remainingListItems > 0 {
		p.byteCodes = append(p.byteCodes, vm.SetList(tableStackIndex, remainingListItems))
	}

	p.byteCodes[newTableByteCodeIndex] = vm.NewTableByteCode(tableStackIndex, listCount, tableCount)
	return newLocalExpression(tableStackIndex), nil
}

func (p *Parser) loadVar(destination byte, identifier string) {
	if pos, ok := p.localsIndex[identifier]; ok {
		p.byteCodes = append(p.byteCodes, vm.Move(destination, pos))
		return
	}

	p.byteCodes = append(p.byteCodes, vm.GetGlobal(destination, p.constants.addString(identifier)))
}

func (p *Parser) newError(inner error) *Error {
	return &Error{inner, p.lexer.Cursor()}
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
	expressionIndex
	expressionIndexField
	expressionIndexInt
	expressionCall
)

type expression struct {
	expressionType expressionType
	inner          any
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

func newIndexExpression(tableStackIndex, keyIndex byte) expression {
	return expression{expressionIndex, [2]byte{tableStackIndex, keyIndex}}
}

func newIndexFieldExpression(tableStackIndex, keyConstIndex byte) expression {
	return expression{expressionIndexField, [2]byte{tableStackIndex, keyConstIndex}}
}

func newIndexIntExpression(tableStackIndex, integer byte) expression {
	return expression{expressionIndexInt, [2]byte{tableStackIndex, integer}}
}

func newCallExpression() expression {
	return expression{expressionCall, nil}
}
