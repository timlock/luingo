package vm

import (
	"fmt"
)

type vmFunc func(*VM) int

type OpCode byte

const (
	getGloal OpCode = iota
	loadConst
	call
)

type ByteCode struct {
	opCode OpCode
	args   [3]byte
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

type Value struct {
	Type  Type
	Inner any
}

type Type = int

const (
	StringType Type = iota
	NumberType
	FunctionType
)

func String(value string) Value {
	return Value{StringType, value}
}

func Function(fn vmFunc) Value {
	return Value{FunctionType, fn}
}

type VM struct {
	globals map[string]Value
	stack   []Value
}

func NewVM(globals map[string]Value) *VM {
	return &VM{globals: globals}
}

func (v *VM) Execute(constants []Value, byteCodes []ByteCode) error {
	for _, byteCode := range byteCodes {
		switch byteCode.opCode {
		case call:
			stackIndex := byteCode.args[0]
			stackItem := v.stack[stackIndex]
			if stackItem.Type != FunctionType {
				return fmt.Errorf("expected %v. stack item to be a function but it is of type %v", stackIndex, stackItem.Type)
			}

			function := stackItem.Inner.(vmFunc)
			_ = function(v)

		case getGloal:
			globalIndex := byteCode.args[1]
			constant := constants[globalIndex]
			if constant.Type != StringType {
				return fmt.Errorf("expected %v. constant to be a global but constant is of type %v", globalIndex, constant.Type)
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

		default:
			panic(fmt.Sprintf("unexpected vm.OpCode: %#v", byteCode.opCode))
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
	stackItem := vm.stack[1]
	fmt.Printf("%v\n", stackItem.Inner)
	return 0
}
