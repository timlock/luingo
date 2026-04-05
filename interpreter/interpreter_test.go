package interpreter

import (
	"context"
	"log/slog"
	"luingo/logging"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	testCases := []struct {
		desc     string
		filePath string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			desc:     "print.lua",
			filePath: path.Join("testdata", "print.lua"),
			wantErr:  assert.NoError,
		},
		{
			desc:     "locals.lua",
			filePath: path.Join("testdata", "locals.lua"),
			wantErr:  assert.NoError,
		},
		{
			desc:     "assign.lua",
			filePath: path.Join("testdata", "assign.lua"),
			wantErr:  assert.NoError,
		},
		{
			desc:     "table.lua",
			filePath: path.Join("testdata", "table.lua"),
			wantErr:  assert.NoError,
		},
		{
			desc:     "prefixexp.lua",
			filePath: path.Join("testdata", "prefixexp.lua"),
			wantErr:  assert.NoError,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			input, err := os.ReadFile(tC.filePath)
			require.NoError(t, err)

			interpreter := NewInterpreter(string(input))

			logger := slog.New(slog.NewTextHandler(
				os.Stderr,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			))
			ctx := logging.WithLogger(context.Background(), logger)
			
			err = interpreter.Execute(ctx)
			tC.wantErr(t, err)
		})
	}
}
