package vm

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"luingo/logging"
	"maps"
	"slices"
	"strings"
)

type vmFunc func(*VM) int

type VM struct {
	globals   map[string]Value
	stack     []Value
	funcIndex int

	out io.Writer
}

func NewVM(globals map[string]Value, stdOut io.Writer) *VM {
	return &VM{globals: globals, out: stdOut}
}

func (v *VM) Execute(ctx context.Context, constants []Value, byteCodes []ByteCode) error {
	logger := logging.Logger(ctx)

	for byteCodeIndex, byteCode := range byteCodes {

		var stringBuilder strings.Builder
		if err := v.step(byteCode, constants); err != nil {
			return fmt.Errorf("executing %+v: %w", byteCode, err)
		}

		stringBuilder.WriteString("Stack: ")
		for stackIndex, value := range v.stack {
			fmt.Fprintf(&stringBuilder, "%v=[%v] ", stackIndex, value)
		}

		logger.Debug(fmt.Sprintf("Step %v. %+v %v", byteCodeIndex, byteCode, stringBuilder.String()))
	}

	return nil
}

func (v *VM) step(byteCode ByteCode, constants []Value) error {
	switch byteCode.opCode {
	case OpCodeCall:
		stackIndex := byteCode.args[0]
		v.funcIndex = int(stackIndex)

		stackItem := v.stack[stackIndex]
		if stackItem.valueType != TypeFunction {
			return fmt.Errorf("expected %v. stack item to be a function but it is of type %v", stackIndex, stackItem.valueType)
		}

		function := stackItem.inner.(vmFunc)
		_ = function(v)

	case OpCodeGetGlobal:
		globalIndex := byteCode.args[1]
		constant := constants[globalIndex]
		if constant.valueType != TypeString {
			return fmt.Errorf("expected %v constant to be a global but constant is of type %v", globalIndex, constant.valueType)
		}

		globalName := constant.inner.(string)

		global, ok := v.globals[globalName]
		if !ok {
			global = NewNil()
		}

		stackIndex := byteCode.args[0]

		v.setStack(int(stackIndex), global)

	case OpCodeSetGlobal:
		globalIndex := byteCode.args[0]
		constant := constants[globalIndex]
		if constant.valueType != TypeString {
			return fmt.Errorf("expected %v constant to be a global but constant is of type %v", globalIndex, constant.valueType)
		}

		stackIndex := byteCode.args[1]
		v.globals[constant.String()] = v.stack[stackIndex]

	case OpCodeSetGlobalGlobal:
		globalIndex := byteCode.args[0]
		constant := constants[globalIndex]
		if constant.valueType != TypeString {
			return fmt.Errorf("expected %v constant to be a global but constant is of type %v", globalIndex, constant.valueType)
		}

		rhGlobalIndex := byteCode.args[1]
		rhConstant := constants[rhGlobalIndex]
		if rhConstant.valueType != TypeString {
			return fmt.Errorf("expected %v constant to be a global but constant is of type %v", globalIndex, rhConstant.valueType)
		}
		rhGlobal, ok := v.globals[rhConstant.String()]
		if !ok {
			rhGlobal = NewNil()
		}
		v.globals[constant.String()] = rhGlobal

	case OpCodeSetGlobalConst:
		globalIndex := byteCode.args[0]
		constant := constants[globalIndex]
		if constant.valueType != TypeString {
			return fmt.Errorf("expected %v constant to be a global but constant is of type %v", globalIndex, constant.valueType)
		}

		constIndex := byteCode.args[1]
		v.globals[constant.String()] = constants[constIndex]

	case OpCodeLoadConst:
		stackIndex := byteCode.args[0]
		constIndex := byteCode.args[1]

		v.setStack(int(stackIndex), constants[constIndex])

	case OpCodeLoadNil:
		stackIndex := byteCode.args[0]
		v.setStack(int(stackIndex), NewNil())

	case OpCodeLoadBool:
		stackIndex := byteCode.args[0]
		isTrue := byteCode.args[1] == 1
		v.setStack(int(stackIndex), NewBoolean(isTrue))

	case OpCodeLoadInt:
		stackIndex := byteCode.args[0]

		integer := binary.BigEndian.Uint16(byteCode.args[1:])

		v.setStack(int(stackIndex), NewInteger(int64(integer)))

	case OpCodeMove:
		destinationIndex := byteCode.args[0]
		sourceIndex := byteCode.args[1]
		v.setStack(int(destinationIndex), v.stack[sourceIndex])

	case OpCodeNewTable:
		stackIndex := byteCode.args[0]
		listSize := byteCode.args[1]
		tableSize := byteCode.args[2]
		v.setStack(int(stackIndex), NewTable(&Table{make([]Value, 0, listSize), make(map[Value]Value, tableSize)}))

	case OpCodeSetTable:
		tableStackIndex := byteCode.args[0]
		keyStackIndex := byteCode.args[1]
		valueStackIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		key := v.stack[keyStackIndex]
		value := v.stack[valueStackIndex]
		table.Put(key, value)

	case OpCodeSetTableConst:
		tableStackIndex := byteCode.args[0]
		keyStackIndex := byteCode.args[1]
		valueConstIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		key := v.stack[keyStackIndex]
		value := constants[valueConstIndex]
		table.Put(key, value)

	case OpCodeSetField:
		tableStackIndex := byteCode.args[0]
		keyConstIndex := byteCode.args[1]
		valueStackIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		key := constants[keyConstIndex]
		value := v.stack[valueStackIndex]
		table.Put(key, value)

	case OpCodeSetFieldConst:
		tableStackIndex := byteCode.args[0]
		keyConstIndex := byteCode.args[1]
		valueConstIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		key := constants[keyConstIndex]
		value := constants[valueConstIndex]
		table.Put(key, value)

	case OpCodeSetInt:
		tableStackIndex := byteCode.args[0]
		listIndex := byteCode.args[1]
		valueStackIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		value := v.stack[valueStackIndex]
		table.Set(int64(listIndex), value)

	case OpCodeSetIntConst:
		tableStackIndex := byteCode.args[0]
		listIndex := byteCode.args[1]
		valueConstIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		value := constants[valueConstIndex]
		table.Set(int64(listIndex), value)

	case OpCodeSetList:
		tableStackIndex := byteCode.args[0]
		listSize := byteCode.args[1]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}

		for i := tableStackIndex + 1; i < tableStackIndex+1+listSize; i++ {
			table.Add(v.stack[i])
			v.stack[i] = Value{}
		}

	case OpCodeGetTable:
		destination := byteCode.args[0]
		tableStackIndex := byteCode.args[1]
		keyStackIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		keyValue := v.stack[keyStackIndex]

		v.setStack(int(destination), table.Get(keyValue))

	case OpCodeGetInt:
		destination := byteCode.args[0]
		tableStackIndex := byteCode.args[1]
		listIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}

		v.setStack(int(destination), table.At(int64(listIndex)))

	case OpCodeGetField:
		destination := byteCode.args[0]
		tableStackIndex := byteCode.args[1]
		keyConstIndex := byteCode.args[2]

		table, err := v.getTable(tableStackIndex)
		if err != nil {
			return err
		}
		keyValue := constants[keyConstIndex]

		v.setStack(int(destination), table.Get(keyValue))

	default:
		panic(fmt.Sprintf("unexpected vm.OpCode: %#v", byteCode.opCode))
	}

	return nil
}

func (v *VM) setStack(index int, value Value) {
	for i := len(v.stack); i <= index; i++ {
		v.stack = append(v.stack, Value{})
	}

	v.stack[index] = value
}

func (v *VM) getTable(index byte) (*Table, error) {
	tableValue := v.stack[index]
	if tableValue.valueType != TypeTable {
		return nil, fmt.Errorf("expected stack value at %v to be a Table but it is of type %v", index, tableValue.valueType)
	}

	return tableValue.inner.(*Table), nil
}

type stack struct {
	inner []Value
}

func Print(vm *VM) int {
	stackItem := vm.stack[vm.funcIndex+1]
	fmt.Fprintf(vm.out, "%v\n", stackItem)
	return 0
}

type OpCode byte

//go:generate go tool stringer -type=OpCode -trimprefix=OpCode
const (
	OpCodeGetGlobal OpCode = iota
	OpCodeSetGlobal
	OpCodeSetGlobalConst
	OpCodeSetGlobalGlobal
	OpCodeLoadConst
	OpCodeCall
	OpCodeLoadNil
	OpCodeLoadBool
	OpCodeLoadInt
	OpCodeMove
	OpCodeNewTable
	OpCodeSetTable
	OpCodeSetTableConst
	OpCodeSetField
	OpCodeSetFieldConst
	OpCodeSetInt
	OpCodeSetIntConst
	OpCodeSetList
	OpCodeGetTable
	OpCodeGetField
	OpCodeGetInt
)

type ByteCode struct {
	opCode OpCode
	args   [3]byte
}

func (b ByteCode) String() string {
	return fmt.Sprintf("%v(%v,%v,%v)", b.opCode, b.args[0], b.args[1], b.args[2])
}

func GetGlobal(stackIndex, globalIndex byte) ByteCode {
	return ByteCode{OpCodeGetGlobal, [3]byte{stackIndex, globalIndex}}
}

func LoadConst(stackIndex, constIndex byte) ByteCode {
	return ByteCode{OpCodeLoadConst, [3]byte{stackIndex, constIndex}}
}

func Call(stackIndex, parameters byte) ByteCode {
	return ByteCode{OpCodeCall, [3]byte{stackIndex, parameters}}
}

func LoadNil(stackIndex byte) ByteCode {
	return ByteCode{OpCodeLoadNil, [3]byte{stackIndex}}
}

func LoadBool(stackIndex byte, value bool) ByteCode {
	var byteValue byte = 0
	if value {
		byteValue = 1
	}

	return ByteCode{OpCodeLoadBool, [3]byte{stackIndex, byteValue}}
}

func LoadInt(stackIndex byte, value int16) ByteCode {
	bytes := [3]byte{stackIndex, 0, 0}
	binary.BigEndian.PutUint16(bytes[1:], uint16(value))
	return ByteCode{OpCodeLoadInt, bytes}
}

func Move(stackIndex, localsIndex byte) ByteCode {
	return ByteCode{OpCodeMove, [3]byte{stackIndex, localsIndex}}
}

func SetGlobalConst(globalIndex, constIndex byte) ByteCode {
	return ByteCode{OpCodeSetGlobalConst, [3]byte{globalIndex, constIndex}}
}

func SetGlobal(globalIndex, stackIndex byte) ByteCode {
	return ByteCode{OpCodeSetGlobal, [3]byte{globalIndex, stackIndex}}
}

func SetGlobalGlobal(globalIndex, constIndex byte) ByteCode {
	return ByteCode{OpCodeSetGlobalGlobal, [3]byte{globalIndex, constIndex}}
}

func NewTableByteCode(tableStackIndex, listSize, tableSize byte) ByteCode {
	return ByteCode{OpCodeNewTable, [3]byte{tableStackIndex, listSize, tableSize}}
}

func SetTable(tableStackIndex, keyStackIndex, valueStackIndex byte) ByteCode {
	return ByteCode{OpCodeSetTable, [3]byte{tableStackIndex, keyStackIndex, valueStackIndex}}
}

func SetTableConst(tableStackIndex, keyStackIndex, valueConstIndex byte) ByteCode {
	return ByteCode{OpCodeSetTableConst, [3]byte{tableStackIndex, keyStackIndex, valueConstIndex}}
}

func SetField(tableStackIndex, keyConstIndex, valueStackIndex byte) ByteCode {
	return ByteCode{OpCodeSetField, [3]byte{tableStackIndex, keyConstIndex, valueStackIndex}}
}

func SetInt(tableStackIndex, integer, valueStackIndex byte) ByteCode {
	return ByteCode{OpCodeSetInt, [3]byte{tableStackIndex, integer, valueStackIndex}}
}

func SetIntConst(tableStackIndex, integer, valueConstIndex byte) ByteCode {
	return ByteCode{OpCodeSetIntConst, [3]byte{tableStackIndex, integer, valueConstIndex}}
}

func SetFieldConst(tableStackIndex, keyConstIndex, valueConstIndex byte) ByteCode {
	return ByteCode{OpCodeSetFieldConst, [3]byte{tableStackIndex, keyConstIndex, valueConstIndex}}
}

func SetList(tableStackIndex, length byte) ByteCode {
	return ByteCode{OpCodeSetList, [3]byte{tableStackIndex, length}}
}

func GetTable(stackIndex, tableStackIndex, keyStackIndex byte) ByteCode {
	return ByteCode{OpCodeGetTable, [3]byte{stackIndex, tableStackIndex, keyStackIndex}}
}

func GetField(stackIndex, tableStackIndex, keyConstIndex byte) ByteCode {
	return ByteCode{OpCodeGetField, [3]byte{stackIndex, tableStackIndex, keyConstIndex}}
}

func GetInt(stackIndex, tableStackIndex, integer byte) ByteCode {
	return ByteCode{OpCodeGetInt, [3]byte{stackIndex, tableStackIndex, integer}}
}

type Value struct {
	valueType Type
	inner     any //TODO store basic types in separate variable
}

func (v Value) String() string {
	switch v.valueType {
	case TypeFunction:
		return "function"
	default:
		return fmt.Sprint(v.inner)
	}
}

//go:generate go tool stringer -type=Type -trimprefix Type

type Type int

const (
	TypeString Type = iota
	TypeFloat
	TypeInteger
	TypeFunction
	TypeBoolean
	TypeNil
	TypeTable
)

func NewNil() Value {
	return Value{TypeNil, nil}
}

func NewString(value string) Value {
	return Value{TypeString, value}
}

func NewFuntion(fn vmFunc) Value {
	return Value{TypeFunction, fn}
}

func NewInteger(value int64) Value {
	return Value{TypeInteger, value}
}

func NewFloat(value float64) Value {
	return Value{TypeFloat, value}
}

func NewBoolean(value bool) Value {
	return Value{TypeBoolean, value}
}

func NewTable(value *Table) Value {
	return Value{TypeTable, value}
}

type Table struct {
	array   []Value
	hashMap map[Value]Value
}

func (t *Table) String() string {
	var stringBuilder strings.Builder

	stringBuilder.WriteString("Table{")

	for i, value := range t.array {
		fmt.Fprintf(&stringBuilder, "%v=%v", i, value)
		if i < len(t.array)-1 {
			stringBuilder.WriteRune(',')
		}
	}

	if len(t.array) > 0 && len(t.hashMap) > 0 {
		stringBuilder.WriteRune(',')
	}

	sortedKeys := slices.SortedFunc(maps.Keys(t.hashMap), func(a, b Value) int {
		return strings.Compare(a.String(), b.String())
	})
	for _, key := range sortedKeys {
		fmt.Fprintf(&stringBuilder, "%v=%v", key, t.hashMap[key])
		stringBuilder.WriteRune(',')
	}

	str := stringBuilder.String()
	str = strings.TrimSuffix(str, ",")
	str += "}"

	return str
}

func (t *Table) Get(key Value) Value {
	if key.inner == TypeInteger {
		index := key.inner.(int64)
		return t.At(index)
	}

	value, ok := t.hashMap[key]
	if !ok {
		return NewNil()
	}

	return value
}

func (t *Table) At(index int64) Value {
	if int64(len(t.array)) <= index {
		return NewNil()
	}

	return t.array[index]
}

func (t *Table) Put(key, value Value) {
	if key.inner == TypeInteger {
		index := key.inner.(int64)
		t.Set(index, value)
		return
	}

	t.hashMap[key] = value
}

func (t *Table) Set(index int64, value Value) {
	for index > int64(len(t.array)) {
		t.array = append(t.array, NewNil())
	}

	t.array = append(t.array, value)
}

func (t *Table) Add(value Value) {
	if len(t.array) == 0 {
		t.array = append(t.array, NewNil())
	}
	t.array = append(t.array, value)
}
