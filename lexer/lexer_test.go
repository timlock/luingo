package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	testCases := []struct {
		desc    string
		input   string
		want    []Token
		wantErr assert.ErrorAssertionFunc
	}{
		{
			desc:  "hello world",
			input: "print \"hello world\"",
			want: []Token{
				{tokenType: Identifier, str: "print"},
				{tokenType: String, str: "hello world"},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			lexer := NewLexer(tC.input)
			got, err := lexer.All()
			tC.wantErr(t, err)
			assert.Equal(t, tC.want, got)
		})
	}
}
