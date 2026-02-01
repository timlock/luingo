package main

import (
	"fmt"
	"luingo/interpreter"
	"os"
	"time"
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

	interpreter := interpreter.NewInterpreter(string(bytes))
	
	start := time.Now()
	if err := interpreter.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)	
	}

	fmt.Printf("Execution took %v", time.Since(start))
	
}
