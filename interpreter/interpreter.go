package interpreter

import (
	"fmt"
	"luingo/parser"
	"luingo/vm"
)

var globals = map[string]vm.Value{
	"print": vm.NewFuntion(vm.Print),
}

type Interpreter struct {
	parser *parser.Parser
	vm     *vm.VM
}

func NewInterpreter(code string) Interpreter {
	return Interpreter{
		parser.NewParser(code),
		vm.NewVM(globals),
	}
}

func (i Interpreter) Execute() error {
	constants, byteCodes, err := i.parser.Parse()
	if err != nil {
		return fmt.Errorf("parsing content: %w", err)
	}

	for _, constant := range constants {
		fmt.Printf("constant: %+v\n", constant)
	}
	for _, byteCode := range byteCodes {
		fmt.Printf("byte code: %v\n", byteCode)
	}

	err = i.vm.Execute(constants, byteCodes)
	if err != nil {
		fmt.Printf("Executing byte code: %v\n", err)
	}

	return nil
}
