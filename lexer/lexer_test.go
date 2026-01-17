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
				{Type: Identifier, Str: "print"},
				{Type: String, Str: "hello world"},
			},
			wantErr: assert.NoError,
		},
		{
			desc:  "1+1",
			input: "1+1",
			want: []Token{
				{Type: Number, Number: 1},
				{Type: Plus},
				{Type: Number, Number: 1},
			},
			wantErr: assert.NoError,
		},
		{
			desc:  "1 + 1",
			input: "1 + 1",
			want: []Token{
				{Type: Number, Number: 1},
				{Type: Plus},
				{Type: Number, Number: 1},
			},
			wantErr: assert.NoError,
		},
		{
			desc:  "1 - 1",
			input: "1 \n-\n 1",
			want: []Token{
				{Type: Number, Number: 1},
				{Type: Minus},
				{Type: Number, Number: 1},
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
