package main

import (
	"fmt"
	"luingo/parser"
	"luingo/vm"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("input file missing")
		return
	}
	bytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("reading file: %v \n", err)
		return
	}

	parser := parser.NewParser(string(bytes))

	constants, byteCodes, err := parser.Parse()
	if err != nil {
		fmt.Printf("parsing content: %v \n", err)
		return
	}

	for _, constant := range constants {
		fmt.Printf("constant: %+v\n", constant)
	}
	for _, byteCode := range byteCodes{
		fmt.Printf("byte code: %v\n", byteCode)
	}

	globals := map[string]vm.Value{
		"print": vm.NewFuntion(vm.Print),
	}
	virtualMachine := vm.NewVM(globals)

	err = virtualMachine.Execute(constants, byteCodes)
	if err != nil {
		fmt.Printf("Executing byte code: %v\n", err)
	}
}
