package interpreter

import (
	"context"
	"fmt"
	"io"
	"luingo/logging"
	"luingo/parser"
	"luingo/vm"
)

var Globals = map[string]vm.Value{
	"print": vm.NewFuntion(vm.Print),
}

type Interpreter struct {
	parser *parser.Parser
	vm     *vm.VM
}

func NewInterpreter(code string, stdOut io.Writer, globals map[string]vm.Value) Interpreter {
	if globals == nil {
		globals = Globals
	}
	return Interpreter{
		parser.NewParser(code),
		vm.NewVM(globals, stdOut),
	}
}

func (i Interpreter) Execute(ctx context.Context) error {
	logger := logging.Logger(ctx)
	constants, byteCodes, err := i.parser.Parse()
	if err != nil {
		return fmt.Errorf("parsing content: %w", err)
	}

	for constantIndex, constant := range constants {
		logger.Debug(fmt.Sprintf("constant: %v=%+v", constantIndex, constant))
	}

	err = i.vm.Execute(ctx, constants, byteCodes)
	if err != nil {
		return fmt.Errorf("Executing byte code: %v\n", err)
	}

	return nil
}
