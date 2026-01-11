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
	tokenType TokenType
	str       string
	number    float64
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
	Number
	String
	Identifier
	LineComment
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
	case Number:
		return "number"
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

type UnexpectedRuneError struct {
	r   rune
	pos int
}

func newUnexpectedRuneError(r rune, pos int) *UnexpectedRuneError {
	return &UnexpectedRuneError{r, pos}
}

func (u *UnexpectedRuneError) Error() string {
	return fmt.Sprintf("unexpected rune '%v' at position %v", u.r, u.pos)
}

type Lexer struct {
	input  *strings.Reader
	buffer strings.Builder
}

func NewLexer(input string) *Lexer {
	return &Lexer{strings.NewReader(input), strings.Builder{}}
}

func (l *Lexer) skipWithespace() (rune, error) {
	var (
		next rune
		err  error
	)
	for {
		next, _, err = l.input.ReadRune()
		if err != nil {
			return 0, fmt.Errorf("reading next rune: %w", err)
		}
		if !unicode.IsSpace(next) {
			break
		}
	}

	l.buffer.WriteRune(next)

	return next, nil
}

func (l *Lexer) readRune() (rune, error) {
	next, _, err := l.input.ReadRune()
		if err != nil {
			return 0, fmt.Errorf("reading next rune: %w", err)
	}

	l.buffer.WriteRune(next)

	return next, nil
}

func (l *Lexer) peekRune() (rune, error) {
	next, _, err := l.input.ReadRune()
	if err != nil {
		return 0, fmt.Errorf("reading next rune: %w", err)
	}

	if err := l.input.UnreadRune(); err != nil {
		return 0, fmt.Errorf("unreading last rune: %w", err)
	}

	return next, nil
}

func (l *Lexer) skipRune() error {
	_, err := l.input.Seek(1, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("reading rune: %w", err)
	}

	return nil
}

func (l *Lexer) takeBuffer() string {
	buffer := l.buffer.String()
	l.buffer.Reset()
	return buffer
}

func (l *Lexer) readLine() (string, error) {
	for {
		peek, err := l.peekRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return l.buffer.String(), nil
			}

			return "", fmt.Errorf("peeking next rune: %w", err)
		}

		if peek == '\r' || peek == '\n' {

			if err := l.skipRune(); err != nil {
				return "", fmt.Errorf("skipping next rune: %w", err)
			}

			break
		}

		if _, err = l.readRune(); err != nil {
			return "", err
		}
	}

	peeked, err := l.peekRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return l.buffer.String(), nil
		}

		return "", fmt.Errorf("peeking next rune: %w", err)
	}

	if peeked == '\r' || peeked == '\n' {
		if err := l.skipRune(); err != nil {
			return "", fmt.Errorf("skipping next rune: %w", err)
		}
	}

	return l.buffer.String(), nil
}

func (l *Lexer) lastRune() (rune, bool) {
	if l.buffer.Len() == 0 {
		return 0, false
	}

	current := []rune(l.buffer.String())
	return current[len(current)-1], true
}

func (l *Lexer) All() ([]Token, error) {
	var tokens []Token

	for {
		token, err := l.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("reading next token: %w", err)
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (l *Lexer) Next() (Token, error) {
	r, err := l.skipWithespace()
	if err != nil {
		return Token{}, err
	}

	defer l.buffer.Reset()

	switch r {
	case '+':
		return Token{tokenType: Plus}, nil
	case '-':
		// TODO comment
		return Token{tokenType: Minus}, nil
	case '"', '\'':
		raw, err := l.readString()
		if err != nil {
			return Token{}, fmt.Errorf("reading string: %w", err)
		}

		return Token{tokenType: String, str: raw}, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		number, _, err := l.readNumber()
		if err != nil {
			return Token{}, fmt.Errorf("reading number: %w", err)
		}

		return Token{tokenType: Number, number: number}, nil
	}

	identifier, err := l.readIdentifier()
	if err != nil {
		return Token{}, fmt.Errorf("reading identifier: %w", err)
	}

	tokenType, err := FromString(identifier)
	if err == nil {
		return Token{tokenType: tokenType}, nil
	}

	return Token{tokenType: Identifier, str: identifier}, nil
}

func (l *Lexer) readString() (string, error) {
	delimiter, ok := l.lastRune()
	if !ok {
		panic("should not call readString() without reading a single quote or doulbe quote first")
	}

	for {
		r, err := l.readRune()
		if err != nil {
			return "", fmt.Errorf("reading next rune: %w", err)
		}

		if r == delimiter {
			str := strings.TrimPrefix(l.takeBuffer(), string(delimiter))
			return strings.TrimSuffix(str, string(delimiter)), nil
		}
	}
}

func (l *Lexer) readIdentifier() (string, error) {
	for {
		r, err := l.peekRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return "", fmt.Errorf("peeking next rune: %w", err)
		}

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}

		if _, err = l.readRune(); err != nil {
			return "", fmt.Errorf("reading next rune: %w", err)
		}

	}

	return l.takeBuffer(), nil
}

func (l *Lexer) ReadRunes() int {
	return int(l.input.Size()) - l.input.Len()
}

func (l *Lexer) readNumber() (float64, string, error) {
	firstPeek, err := l.peekRune()
	if err != nil {
		return 0, "", fmt.Errorf("peeking next rune: %w", err)
	}

	secondPeek, err := l.peekRune()
	if err != nil {
		return 0, "", fmt.Errorf("peeking next rune: %w", err)
	}

	if firstPeek == '0' && (secondPeek == 'x' || secondPeek == 'X') {
		// the 0x part of a hex number cant be parsed by strconv
		_, err := l.input.Seek(2, io.SeekCurrent)
		if err != nil {
			return 0, "", fmt.Errorf("skipping next two runes: %w", err)
		}

		for {
			next, err := l.readRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				return 0, "", fmt.Errorf("reading next rune: %w", err)
			}

			if !unicode.IsDigit(next) && next != '.' && !unicode.IsLetter(next) {
				if err := l.input.UnreadRune(); err != nil {
					return 0, "", fmt.Errorf("unreading last rune: %w", err)
				}

				break
			}

			if next == 'e' || next == 'E' || next == 'a' || next == 'A' {
				if nextPeek, err := l.peekRune(); err == nil && (nextPeek == '+' || nextPeek == '-') {
					if _, err = l.readRune(); err != nil {
						return 0, "", fmt.Errorf("reading next rune: %w", err)
					}
				}
			}
		}

	} else {
		for {
			next, err := l.readRune()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				return 0, "", fmt.Errorf("reading next rune: %w", err)
			}

			if !unicode.IsDigit(next) && next != '.' {
				if err := l.input.UnreadRune(); err != nil {
					return 0, "", fmt.Errorf("unreading last rune: %w", err)
				}

				break
			}
		}
	}

	raw := l.takeBuffer()
	number, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, "", fmt.Errorf("parsing number from %v: %w", raw, err)
	}

	return number, raw, nil
}
