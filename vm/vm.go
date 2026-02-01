package vm

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type vmFunc func(*VM) int

type OpCode byte

const (
	getGloal OpCode = iota
	loadConst
	call
	loadNil
	loadBool
	loadInt
	move
)

func (o OpCode) String() string {
	switch o {
	case call:
		return "Call"
	case getGloal:
		return "GetGlobal"
	case loadBool:
		return "LoadBool"
	case loadConst:
		return "LoadConst"
	case loadInt:
		return "LoadInt"
	case loadNil:
		return "LoadNil"
	case move:
		return "Move"
	default:
		panic(fmt.Sprintf("unexpected vm.OpCode: %#v", o))
	}
}

type ByteCode struct {
	opCode OpCode
	args   [3]byte
}

func (b ByteCode) String() string {
	return fmt.Sprintf("%v(%v,%v,%v)", b.opCode, b.args[0], b.args[1], b.args[2])
}

func GetGlobal(stackIndex, globalIndex byte) ByteCode {
	return ByteCode{getGloal, [3]byte{stackIndex, globalIndex, 0}}
}

func LoadConst(stackIndex, constIndex byte) ByteCode {
	return ByteCode{loadConst, [3]byte{stackIndex, constIndex, 0}}
}

func Call(stackIndex, parameters byte) ByteCode {
	return ByteCode{call, [3]byte{stackIndex, parameters, 0}}
}

func LoadNil(stackIndex byte) ByteCode {
	return ByteCode{loadNil, [3]byte{stackIndex, 0, 0}}
}

func LoadBool(stackIndex byte, value bool) ByteCode {
	var byteValue byte = 0
	if value {
		byteValue = 1
	}

	return ByteCode{loadBool, [3]byte{stackIndex, byteValue, 0}}
}

func LoadInt(stackIndex byte, value int16) (ByteCode, error) {
	bytes := [3]byte{stackIndex, 0, 0}
	if _, err := binary.Encode(bytes[1:], binary.BigEndian, value); err != nil {
		return ByteCode{}, fmt.Errorf("converting int16 %v to 2 bytes: %w", value, err)
	}

	return ByteCode{loadInt, bytes}, nil
}

func Move(stackIndex, localsIndex byte) ByteCode {
	bytes := [3]byte{stackIndex, localsIndex, 0}

	return ByteCode{move, bytes}
}

type Value struct {
	Type  Type
	Inner any
}

func (v Value) String() string {
	switch v.Type {
	case FunctionType:
		return "function"
	default:
		return fmt.Sprint(v.Inner)
	}
}

type Type int

const (
	StringType Type = iota
	FloatType
	IntegerType
	FunctionType
	BooleanType
	NilType
)

func (t Type) String() string {
	switch t {
	case StringType:
		return "String"
	case FloatType:
		return "Float"
	case IntegerType:
		return "Integer"
	case BooleanType:
		return "Boolean"
	case FunctionType:
		return "Function"
	case NilType:
		return "Nil"
	default:
		panic(fmt.Sprintf("unexpected vm.Type: %#v", t))
	}
}

func NewNil() Value {
	return Value{NilType, nil}
}

func NewString(value string) Value {
	return Value{StringType, value}
}

func NewFuntion(fn vmFunc) Value {
	return Value{FunctionType, fn}
}

func NewInteger(value int64) Value {
	return Value{IntegerType, value}
}

func NewFloat(value float64) Value {
	return Value{FloatType, value}
}

func NewBoolean(value bool) Value {
	return Value{BooleanType, value}
}

type VM struct {
	globals   map[string]Value
	stack     []Value
	funcIndex int
}

func NewVM(globals map[string]Value) *VM {
	return &VM{globals: globals}
}

func (v *VM) Execute(constants []Value, byteCodes []ByteCode) error {
	for _, byteCode := range byteCodes {
		switch byteCode.opCode {
		case call:
			stackIndex := byteCode.args[0]
			v.funcIndex = int(stackIndex)

			stackItem := v.stack[stackIndex]
			if stackItem.Type != FunctionType {
				return fmt.Errorf("expected %v. stack item to be a function but it is of type %v", stackIndex, stackItem.Type)
			}

			function, ok := stackItem.Inner.(vmFunc)
			if !ok {
				panic(fmt.Sprintf("Value type is %v but inner is %v", stackItem.Type, reflect.TypeOf(function)))
			}
			_ = function(v)

		case getGloal:
			globalIndex := byteCode.args[1]
			constant := constants[globalIndex]
			if constant.Type != StringType {
				return fmt.Errorf("expected %v constant to be a global but constant is of type %v", globalIndex, constant.Type)
			}

			globalName := constant.Inner.(string)
			global, ok := v.globals[globalName]
			if !ok {
				return fmt.Errorf("global '%v' does not exist", globalName)
			}

			stackIndex := byteCode.args[0]

			v.setStack(int(stackIndex), global)
		case loadConst:
			stackIndex := byteCode.args[0]
			constIndex := byteCode.args[1]

			v.setStack(int(stackIndex), constants[constIndex])

		case loadNil:
			stackIndex := byteCode.args[0]
			v.setStack(int(stackIndex), NewNil())

		case loadBool:
			stackIndex := byteCode.args[0]
			isTrue := byteCode.args[1] == 1
			v.setStack(int(stackIndex), NewBoolean(isTrue))

		case loadInt:
			stackIndex := byteCode.args[0]

			var integer int16
			_, err := binary.Decode(byteCode.args[1:], binary.BigEndian, &integer)
			if err != nil {
				return fmt.Errorf("decoding integer from ByteCode %+v : %w", byteCode, err)
			}

			v.setStack(int(stackIndex), NewInteger(int64(integer)))
		case move:
			destinationIndex := byteCode.args[0]
			sourceIndex := byteCode.args[1]
			v.setStack(int(destinationIndex), v.stack[sourceIndex])
		default:
			return fmt.Errorf("unexpected vm.OpCode: %v", byteCode.opCode)
		}
	}

	return nil
}

func (v *VM) setStack(index int, value Value) {
	for i := len(v.stack); i <= index; i++ {
		v.stack = append(v.stack, Value{})
	}

	v.stack[index] = value
}

func Print(vm *VM) int {
	stackItem := vm.stack[vm.funcIndex+1]
	fmt.Printf("%v\n", stackItem)
	return 0
}
