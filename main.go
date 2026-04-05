package main

import (
	"context"
	"fmt"
	"log/slog"
	"luingo/interpreter"
	"luingo/logging"
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
	logger := slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{Level: slog.LevelInfo},
	))
	ctx := logging.WithLogger(context.Background(), logger)

	start := time.Now()
	if err := interpreter.Execute(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("Execution took %v", time.Since(start))

}
