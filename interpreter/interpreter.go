package interpreter

import (
	"context"
	"fmt"
	"io"
	"luingo/logging"
	"luingo/parser"
	"luingo/vm"
	"time"
)

var Globals = map[string]vm.Value{
	"print": vm.NewFuntion(vm.Print),
}

type Options struct {
	Globals map[string]vm.Value
	Out     io.Writer
}

type Interpreter struct {
	parser *parser.Parser
	vm     *vm.VM
}

func NewInterpreter(code string, options Options) Interpreter {
	if options.Globals == nil {
		options.Globals = Globals
	}
	if options.Out == nil {
		options.Out = io.Discard
	}

	return Interpreter{
		parser.NewParser(code),
		vm.NewVM(options.Globals, options.Out),
	}
}

func (i Interpreter) Execute(ctx context.Context) error {
	logger := logging.Logger(ctx)
	start := time.Now()
	constants, byteCodes, err := i.parser.Parse()
	if err != nil {
		return fmt.Errorf("parsing content: %w", err)
	}
	logger.Debug("Parsing complete", "duration", time.Since(start))

	for i, constant := range constants {
		logger.Debug(fmt.Sprintf("constant: %v=%+v", i, constant))
	}

	for i, byteCode := range byteCodes {
		logger.Debug(fmt.Sprintf("bytecode: %v=%+v", i, byteCode))
	}

	start = time.Now()

	err = i.vm.Execute(ctx, constants, byteCodes)
	if err != nil {
		return fmt.Errorf("Executing byte code: %v\n", err)
	}

	logger.Debug("Execution complete", "duration", time.Since(start))

	return nil
}
