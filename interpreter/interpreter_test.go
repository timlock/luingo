package interpreter

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrint(t *testing.T) {
	input, err := os.ReadFile(path.Join("testdata", "print.lua"))
	require.NoError(t, err)
	interpreter := NewInterpreter(string(input))
	err = interpreter.Execute()
	assert.NoError(t, err)
}

func TestLocal(t *testing.T) {
	input, err := os.ReadFile(path.Join("testdata", "locals.lua"))
	require.NoError(t, err)

	interpreter := NewInterpreter(string(input))
	err = interpreter.Execute()
	assert.NoError(t, err)
}

func TestAssignment(t *testing.T) {
	input, err := os.ReadFile(path.Join("testdata", "assign.lua"))
	require.NoError(t, err)

	interpreter := NewInterpreter(string(input))
	err = interpreter.Execute()
	assert.NoError(t, err)
}

