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

	constants, byteCode, err := parser.Parse()
	if err != nil {
		fmt.Printf("parsing content: %v \n", err)
		return
	}

	globals := map[string]vm.Value{
		"print": vm.Function(vm.Print),
	}
	virtualMachine := vm.NewVM(globals)

	err = virtualMachine.Execute(constants, byteCode)
	if err != nil {
		fmt.Printf("Executing byte code: %v\n", err)
	}
}
