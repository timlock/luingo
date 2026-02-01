package lexer

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type Token struct {
	Type    TokenType
	Str     string
	Float   float64
	Integer int64
}

type TokenType int

const (
	And TokenType = iota + 1
	Break
	Do
	Else
	ElseIf
	End
	False
	For
	Function
	Global
	Goto
	If
	In
	Local
	Nil
	Not
	Or
	Repeat
	Return
	Then
	True
	Until
	While
	Plus
	Minus
	Asterisk
	Slash
	Percentage
	Cirumflex
	Hashtag
	Ampersand
	Tilde
	Pipe
	RightShfit
	LeftShift
	EscpaedSlash
	Equal
	NotEqual
	SmallerThan
	GreaterThan
	Smaller
	Greater
	Assign
	OpenBracket
	ClosedBracket
	OpenBrace
	ClosedBrace
	OpenSquareBracket
	ClosedSquareBracket
	DoubleColon
	SemiColon
	Colon
	Comma
	Dot
	DoubleDot
	TrippleDot
	Float
	Integer
	String
	Identifier
	LineComment
	EOF
)

func (t TokenType) String() string {
	switch t {
	case And:
		return "and"
	case Break:
		return "break"
	case Do:
		return "do"
	case Else:
		return "else"
	case ElseIf:
		return "elseif"
	case End:
		return "end"
	case False:
		return "false"
	case For:
		return "for"
	case Function:
		return "function"
	case Global:
		return "global"
	case Goto:
		return "goto"
	case If:
		return "if"
	case In:
		return "in"
	case Local:
		return "local"
	case Nil:
		return "nil"
	case Not:
		return "not"
	case Or:
		return "or"
	case Repeat:
		return "repeat"
	case Return:
		return "return"
	case Then:
		return "then"
	case True:
		return "true"
	case Until:
		return "until"
	case While:
		return "while"
	case Plus:
		return "+"
	case Minus:
		return "-"
	case Asterisk:
		return "*"
	case Slash:
		return "//"
	case Percentage:
		return "%"
	case Cirumflex:
		return "^"
	case Hashtag:
		return "#"
	case Ampersand:
		return "&"
	case Tilde:
		return "~"
	case Pipe:
		return "|"
	case RightShfit:
		return "=<<"
	case LeftShift:
		return "==>"
	case EscpaedSlash:
		return "\\\\"
	case Equal:
		return "=="
	case NotEqual:
		return "~="
	case SmallerThan:
		return "<="
	case GreaterThan:
		return ">="
	case Smaller:
		return "<"
	case Greater:
		return ">"
	case Assign:
		return "="
	case OpenBracket:
		return "("
	case ClosedBracket:
		return ")"
	case OpenBrace:
		return "["
	case ClosedBrace:
		return "]"
	case OpenSquareBracket:
		return "{"
	case ClosedSquareBracket:
		return "}"
	case DoubleColon:
		return "::"
	case SemiColon:
		return ";"
	case Colon:
		return ":"
	case Comma:
		return ","
	case Dot:
		return "."
	case DoubleDot:
		return ".."
	case TrippleDot:
		return "..."
	case String:
		return "string"
	case Identifier:
		return "identifier"
	case LineComment:
		return "LineComment"
	default:
		return "unknown token"
		// return fmt.Sprintf("unknown token '%v'", t)
	}
}

func FromString(value string) (TokenType, error) {
	switch value {
	case "and":
		return And, nil
	case "break":
		return Break, nil
	case "do":
		return Do, nil
	case "else":
		return Else, nil
	case "elseif":
		return ElseIf, nil
	case "end":
		return End, nil
	case "false":
		return False, nil
	case "for":
		return For, nil
	case "function":
		return Function, nil
	case "global":
		return Global, nil
	case "goto":
		return Goto, nil
	case "if":
		return If, nil
	case "in":
		return In, nil
	case "local":
		return Local, nil
	case "nil":
		return Nil, nil
	case "not":
		return Not, nil
	case "or":
		return Or, nil
	case "repeat":
		return Repeat, nil
	case "return":
		return Return, nil
	case "then":
		return Then, nil
	case "true":
		return True, nil
	case "until":
		return Until, nil
	case "while":
		return While, nil
	case "+":
		return Plus, nil
	case "-":
		return Minus, nil
	case "*":
		return Asterisk, nil
	case "/":
		return Slash, nil
	case "%":
		return Percentage, nil
	case "^":
		return Cirumflex, nil
	case "#":
		return Hashtag, nil
	case "&":
		return Ampersand, nil
	case "~":
		return Tilde, nil
	case "|":
		return Pipe, nil
	case "=<<":
		return RightShfit, nil
	case "=>>":
		return LeftShift, nil
	case "//":
		return EscpaedSlash, nil
	case "==":
		return Equal, nil
	case "~=":
		return NotEqual, nil
	case "<=":
		return SmallerThan, nil
	case ">=":
		return GreaterThan, nil
	case "<":
		return Smaller, nil
	case ">":
		return Greater, nil
	case "=":
		return Assign, nil
	case "(":
		return OpenBracket, nil
	case ")":
		return ClosedBracket, nil
	case "{":
		return OpenBrace, nil
	case "}":
		return ClosedBrace, nil
	case "[":
		return OpenSquareBracket, nil
	case "]":
		return ClosedSquareBracket, nil
	case "::":
		return DoubleColon, nil
	case ";":
		return SemiColon, nil
	case ":":
		return Colon, nil
	case ",":
		return Comma, nil
	case ".":
		return Dot, nil
	case "..":
		return DoubleDot, nil
	case "...":
		return TrippleDot, nil
	default:
		return 0, fmt.Errorf("unknown symbol %s", value)
	}
}

type Error struct {
	inner  error
	cursor Cursor
}

func NewError(inner error, cursor Cursor) *Error {
	return &Error{inner, cursor}
}

func (l *Error) Error() string {
	return fmt.Sprintf("%+v %v", l.cursor, l.inner)
}

func (e *Error) Unwrap() error {
	return e.inner
}

type Cursor struct {
	line, col int
}

type reader struct {
	inner  *strings.Reader
	Cursor Cursor
}

func NewDiagnosticReader(input string) *reader {
	return &reader{strings.NewReader(input), Cursor{1, 1}}
}

func (d *reader) TakeRune() (rune, bool) {
	next, _, err := d.inner.ReadRune()
	if err != nil {
		return 0, false
	}

	d.Cursor.col++
	if next == '\n' {
		d.Cursor.line++
		d.Cursor.col = 1
	}

	return next, true
}

func (r *reader) PeekRune() (rune, bool) {
	next, _, err := r.inner.ReadRune()
	if err != nil {
		return 0, false
	}
	r.inner.UnreadRune()

	return next, true
}

func (r *reader) SkipRunes(n int64) {
	r.inner.Seek(n, io.SeekCurrent)
}

func (r *reader) Size() int64 {
	return r.inner.Size()
}

func (r *reader) Len() int {
	return r.inner.Len()
}

type Lexer struct {
	input  reader
	buffer strings.Builder
}

func NewLexer(input string) *Lexer {
	return &Lexer{*NewDiagnosticReader(input), strings.Builder{}}
}

func (l *Lexer) Cursor() Cursor {
	return l.input.Cursor
}

func (l *Lexer) skipWithespace() (rune, bool) {
	var (
		next rune
		ok   bool
	)
	for {
		next, ok = l.input.TakeRune()
		if !ok {
			return 0, false
		}

		if !unicode.IsSpace(next) {
			break
		}
	}

	l.buffer.WriteRune(next)

	return next, true
}

func (l *Lexer) readRune() (rune, bool) {
	next, ok := l.input.TakeRune()
	if !ok {
		return 0, false
	}

	l.buffer.WriteRune(next)

	return next, true
}

func (l *Lexer) readIf(want rune) bool {
	got, ok := l.input.PeekRune()
	if !ok {
		return false
	}

	if got != want {
		return false
	}
	_, ok = l.readRune()
	return ok
}

func (l *Lexer) takeBuffer() string {
	buffer := l.buffer.String()
	l.buffer.Reset()
	return buffer
}

func (l *Lexer) readLine() string {
	for {
		peek, ok := l.input.PeekRune()
		if !ok {
			return l.takeBuffer()
		}

		if peek == '\r' || peek == '\n' {

			l.input.SkipRunes(1)
			break
		}

		l.readRune()
	}

	peek, ok := l.input.PeekRune()
	if !ok {
		return l.takeBuffer()
	}

	if peek == '\r' || peek == '\n' {

		l.input.SkipRunes(1)
	}

	return l.takeBuffer()
}

func (l *Lexer) lastRune() (rune, bool) {
	if l.buffer.Len() == 0 {
		return 0, false
	}

	current := []rune(l.buffer.String())
	return current[len(current)-1], true
}

func (l *Lexer) readWhile(matchFn func(rune) bool) {
	for {
		nextRune, ok := l.input.PeekRune()
		if !ok {
			return
		}

		if !matchFn(nextRune) {
			return
		}

		_, ok = l.readRune()
	}
}

func (l *Lexer) All() ([]Token, *Error) {
	var tokens []Token

	for {
		token, err := l.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (l *Lexer) Next() (Token, *Error) {
	r, ok := l.skipWithespace()
	if !ok {
		return Token{}, NewError(io.EOF, l.Cursor())
	}

	defer l.buffer.Reset()

	switch r {
	case '+':
		return Token{Type: Plus}, nil
	case '-':
		ok := l.readIf('-')
		if ok {
			comment := l.readLine()
			return Token{Type: LineComment, Str: comment}, nil
		}
		return Token{Type: Tilde}, nil
	case '*':
		return Token{Type: Asterisk}, nil
	case '/':
		ok := l.readIf('/')
		if ok {
			return Token{Type: EscpaedSlash}, nil
		}
		return Token{Type: Slash}, nil
	case '%':
		return Token{Type: Ampersand}, nil
	case '^':
		return Token{Type: Cirumflex}, nil
	case '#':
		return Token{Type: Hashtag}, nil

	case '&':
		return Token{Type: Ampersand}, nil
	case '|':
		return Token{Type: Pipe}, nil

	case '=':
		ok := l.readIf('=')
		if ok {
			return Token{Type: Equal}, nil
		}
		return Token{Type: Assign}, nil

	case '~':
		ok := l.readIf('=')
		if ok {
			return Token{Type: NotEqual}, nil
		}
		return Token{Type: Tilde}, nil

	case '<':
		ok := l.readIf('=')
		if ok {
			return Token{Type: SmallerThan}, nil
		}
		return Token{Type: Smaller}, nil
	case '>':
		ok := l.readIf('=')
		if ok {
			return Token{Type: GreaterThan}, nil
		}
		return Token{Type: Greater}, nil
	case '(':
		return Token{Type: OpenBracket}, nil
	case ')':
		return Token{Type: ClosedBracket}, nil
	case '{':
		return Token{Type: OpenBrace}, nil
	case '}':
		return Token{Type: ClosedBrace}, nil
	case '[':
		return Token{Type: OpenSquareBracket}, nil
	case ']':
		return Token{Type: ClosedSquareBracket}, nil

	case ':':
		ok := l.readIf(':')
		if ok {
			return Token{Type: DoubleColon}, nil
		}
		return Token{Type: Colon}, nil

	case ';':
		return Token{Type: SemiColon}, nil
	case ',':
		return Token{Type: Comma}, nil
	case '.':
		ok := l.readIf('.')
		if !ok {
			return Token{Type: Dot}, nil
		}

		ok = l.readIf('.')
		if !ok {
			return Token{Type: DoubleDot}, nil
		}

		return Token{Type: TrippleDot}, nil

	case '"', '\'':
		raw, err := l.readString()
		if err != nil {
			return Token{}, err
		}

		return Token{Type: String, Str: raw}, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		token, err := l.readNumber()
		if err != nil {
			return Token{}, NewError(fmt.Errorf("reading number: %w", err), l.Cursor())
		}

		return token, nil
	}

	identifier := l.readIdentifier()
	tokenType, err := FromString(identifier)
	if err == nil {
		return Token{Type: tokenType}, nil
	}

	return Token{Type: Identifier, Str: identifier}, nil
}

func (l *Lexer) ExpectToken(want TokenType) (Token, *Error) {
	token, err := l.Next()
	if err != nil {
		return Token{}, err
	}

	if token.Type != want {
		return Token{}, NewError(fmt.Errorf("want %v got %v", want, token.Type), l.Cursor())
	}

	return token, nil
}

func (l *Lexer) readString() (string, *Error) {
	delimiter, ok := l.lastRune()
	if !ok {
		panic("should not call readString() without reading a single quote or doulbe quote first")
	}

	l.readWhile(func(next rune) bool {
		return next != delimiter && next != '\n'
	})

	if closingDelimiter, ok := l.input.PeekRune(); !ok || closingDelimiter != delimiter {
		return "", NewError(errors.New("cut off string"), l.Cursor())
	}
	l.input.SkipRunes(1)

	str := strings.TrimPrefix(l.takeBuffer(), string(delimiter))
	return str, nil
}

func (l *Lexer) readIdentifier() string {
	l.readWhile(func(next rune) bool {
		return unicode.IsDigit(next) || unicode.IsLetter(next)
	})

	return l.takeBuffer()
}

func (l *Lexer) ReadRunes() int {
	return int(l.input.Size()) - l.input.Len()
}

func (l *Lexer) readNumber() (Token, *Error) {
	lastRead, ok := l.lastRune()
	if !ok {
		panic("should not call readNumber() without reading a digit first")
	}

	isFloat := func() bool {
		isFloat := false
		peeked, ok := l.input.PeekRune()
		if !ok {
			return false
		}

		if lastRead == '0' && (peeked == 'x' || peeked == 'X') {
			// the 0x part of a hex number cant be parsed by strconv
			l.input.SkipRunes(2)

			for {
				peeked, ok := l.input.PeekRune()
				if !ok {
					break
				}

				if !unicode.IsDigit(peeked) &&
					peeked != '.' &&
					!unicode.In(peeked, unicode.ASCII_Hex_Digit) &&
					peeked != 'p' && peeked != 'P' {

					break
				}

				if peeked == '.' || peeked == 'p' || peeked == 'P' {
					isFloat = true
				}

				l.readRune()

				if peeked == 'p' || peeked == 'P' {
					// next rune might be a + or a -
					l.readRune()
				}
			}

		} else if unicode.IsDigit(peeked) {
			for {
				peeked, ok := l.input.PeekRune()
				if !ok {
					break
				}

				if !unicode.IsDigit(peeked) && peeked != '.' && peeked != 'e' && peeked != 'E' {
					break
				}

				if peeked == '.' || peeked == 'e' || peeked == 'E' {
					isFloat = true
				}

				l.readRune()

				if peeked == 'e' || peeked == 'E' {
					// next rune might be a + or a -
					l.readRune()
				}
			}
		}

		return isFloat
	}()

	raw := l.takeBuffer()
	if isFloat {
		number, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return Token{}, NewError(fmt.Errorf("parsing float64 from %v: %w", raw, err), l.Cursor())
		}

		return Token{Type: Float, Float: number}, nil
	}

	number, err := strconv.ParseInt(raw, 10, 64) //TODO different base?
	if err != nil {
		return Token{}, NewError(fmt.Errorf("parsing int64 from %v: %w", raw, err), l.Cursor())
	}

	return Token{Type: Integer, Integer: number}, nil

}
