package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Parser struct {
	r *bufio.Reader
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		r: bufio.NewReader(r),
	}
}

type Request struct {
	Name string
	Args []Value
}

func (p *Parser) ReadValue() (Value, error) {
	commandType, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch commandType {
	case '*':
		return p.parseArray()
	case '$':
		return p.parseBulkString()
	case ':':
		return p.parseInteger()
	case '-':
		return p.parseError()
	case '+':
		return p.parseSimpleString()
	case '_':
		return p.parseNull()
	default:
		return ErrorValue{"ERR unknown RESP type"}, nil
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	lineLen := len(line)
	if lineLen < 2 || line[lineLen-2] != '\r' {
		return "", fmt.Errorf("protocol error: expected \\r\\n")
	}
	return line[:lineLen-2], nil
}

func (p *Parser) parseArray() (Value, error) {
	line, err := readLine(p.r)
	if err != nil {
		return nil, fmt.Errorf("error parse ArrayValue: %w", err)
	}
	count, convErr := strconv.ParseInt(line, 10, 64)
	if convErr != nil {
		return nil, fmt.Errorf("error convert string to int: %w", convErr)
	}
	if count == -1 {
		return NullValue{}, nil // RESP null array
	}
	items := make([]Value, 0, count)
	for range count {
		val, err := p.ReadValue()
		if err != nil {
			return nil, err
		}
		items = append(items, val)
	}
	return ArrayValue{Data: items}, nil
}

func (p *Parser) ReadRequest() (*Request, error) {
	val, err := p.ReadValue()
	if err != nil {
		return nil, err
	}
	arr, ok := val.(ArrayValue)
	if !ok {
		return &Request{}, errors.New("ERR expected array command")
	}
	if len(arr.Data) == 0 {
		return &Request{}, nil
	}

	nameValue := arr.Data[0]
	name := strings.ToUpper(strings.TrimSpace(nameValue.String()))
	args := arr.Data[1:]

	return &Request{
		Name: name,
		Args: args,
	}, nil
}

func (p *Parser) parseSimpleString() (Value, error) {
	line, err := readLine(p.r)
	if err != nil {
		return nil, fmt.Errorf("error parse SimpleStringValue: %w", err)
	}
	return SimpleStringValue{Data: []byte(line)}, nil
}

func (p *Parser) parseBulkString() (Value, error) {
	line, err := readLine(p.r)
	if err != nil {
		return nil, fmt.Errorf("error parse BulkStringValue: %w", err)
	}
	if line == "-1" {
		return NullValue{}, nil
	}
	n, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error convert string to int: %w", err)
	}
	data := make([]byte, n)
	_, err = io.ReadFull(p.r, data)
	if err != nil {
		return nil, err
	}
	crlf := make([]byte, 2)
	_, err = io.ReadFull(p.r, crlf)
	if err != nil {
		return nil, err
	}
	if !(crlf[0] == '\r' && crlf[1] == '\n') {
		return nil, fmt.Errorf("protocol error: expected \\r\\n")
	}
	return BulkStringValue{Data: data}, nil
}

func (p *Parser) parseInteger() (Value, error) {
	line, err := readLine(p.r)
	if err != nil {
		return nil, fmt.Errorf("error parse IntegerValue: %w", err)
	}
	num, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error convert string to int: %w", err)
	}
	return IntegerValue{Value: num}, nil
}

func (p *Parser) parseError() (Value, error) {
	line, err := readLine(p.r)
	if err != nil {
		return nil, fmt.Errorf("error parse ErrorValue: %w", err)
	}

	return ErrorValue{Message: line}, nil
}

func (p *Parser) parseNull() (Value, error) {
	_, err := readLine(p.r)
	if err != nil {
		return nil, fmt.Errorf("error parse NullValue: %w", err)
	}
	return NullValue{}, nil
}

func (p *Parser) Encode(v Value) ([]byte, error) {
	switch v.Type() {
	case TypeSimpleString:
		return []byte("+" + v.String() + "\r\n"), nil
	case TypeError:
		return []byte("-" + v.String() + "\r\n"), nil
	case TypeBulkString:
		data := v.Bytes()
		return []byte("$" + strconv.Itoa(len(data)) + "\r\n" + string(data) + "\r\n"), nil
	case TypeInteger:
		return []byte(":" + v.String() + "\r\n"), nil
	case TypeArray:
		arr, ok := v.(ArrayValue)
		if !ok {
			return nil, fmt.Errorf("expected ArrayValue")
		}
		var buf bytes.Buffer
		buf.WriteString("*" + strconv.Itoa(len(arr.Elements())) + "\r\n")
		for _, item := range arr.Elements() {
			i, err := p.Encode(item)
			if err != nil {
				return nil, err
			}
			buf.Write(i)
		}
		return buf.Bytes(), nil
	case TypeNull:
		return []byte("$-1\r\n"), nil
	default:
		return nil, fmt.Errorf("unknown value type: %s", v.Type())
	}
}
