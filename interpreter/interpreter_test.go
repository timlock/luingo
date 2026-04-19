package interpreter

import (
	"context"
	"log/slog"
	"luingo/logging"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	testCases := []struct {
		desc       string
		filePath   string
		wantOutput []string
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			desc:       "print.lua",
			filePath:   path.Join("testdata", "print.lua"),
			wantOutput: []string{"hello, world!", "<nil>", "false", "123", "123456", "123456"},
			wantErr:    assert.NoError,
		},
		{
			desc:       "locals.lua",
			filePath:   path.Join("testdata", "locals.lua"),
			wantOutput: []string{"hello, local!", "function", "I'm local-print!"},
			wantErr:    assert.NoError,
		},
		{
			desc:       "assign.lua",
			filePath:   path.Join("testdata", "assign.lua"),
			wantOutput: []string{"123", "123", "<nil>", "123", "<nil>", "<nil>"},
			wantErr:    assert.NoError,
		},
		{
			desc:       "table.lua",
			filePath:   path.Join("testdata", "table.lua"),
			wantOutput: []string{"100", "hello", "vvv", "Table{0=<nil>,1=100,2=200,3=300,kkk=vvv,x=hello,y=world}"},
			wantErr:    assert.NoError,
		},
		{
			desc:       "prefixexp.lua",
			filePath:   path.Join("testdata", "prefixexp.lua"),
			wantOutput: []string{"400", "100", "20", "<nil>"},
			wantErr:    assert.NoError,
		},
		{
			desc:     "unary_operations.lua",
			filePath: path.Join("testdata", "unary_operations.lua"),
			wantOutput: []string{"-101","-101","-3.14","-3.14","10","10","true","true","false","false"},
			wantErr: assert.NoError,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			input, err := os.ReadFile(tC.filePath)
			require.NoError(t, err)

			var output strings.Builder

			interpreter := NewInterpreter(string(input), Options{
				Globals,
				&output,
			})

			logger := slog.New(slog.NewTextHandler(
				os.Stderr,
				&slog.HandlerOptions{
					Level: slog.LevelDebug,
					ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
						if a.Key == slog.TimeKey {
							return slog.Attr{
								Key:   a.Key,
								Value: slog.StringValue(a.Value.Time().Format(time.TimeOnly)),
							}
						}

						return a
					},
				},
			))
			ctx := logging.WithLogger(context.Background(), logger)

			err = interpreter.Execute(ctx)
			tC.wantErr(t, err)

			gotOutput := strings.Split(output.String(), "\n")
			gotOutput = gotOutput[:len(gotOutput)-1] // last element is empty
			assert.Equal(t, tC.wantOutput, gotOutput)
		})
	}
}
