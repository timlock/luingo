package interpreter

import (
	"context"
	"fmt"
	"luingo/logging"
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

func (i Interpreter) Execute(ctx context.Context) error {
	logger := logging.Logger(ctx)
	constants, byteCodes, err := i.parser.Parse()
	if err != nil {
		return fmt.Errorf("parsing content: %w", err)
	}

	for _, constant := range constants {
		logger.Debug(fmt.Sprintf("constant: %+v", constant))
	}
	for _, byteCode := range byteCodes {
		logger.Debug(fmt.Sprintf("byte code: %v", byteCode))
	}

	err = i.vm.Execute(ctx, constants, byteCodes)
	if err != nil {
		return fmt.Errorf("Executing byte code: %v\n", err)
	}

	return nil
}
