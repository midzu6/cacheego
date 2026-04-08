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
}

func NewParser() *Parser {
	return &Parser{}
}

type Request struct {
	Name string
	Args []Value
}

func (p *Parser) ReadValue(r io.Reader) (Value, error) {
	br := bufio.NewReader(r)
	commandType, err := br.ReadByte()
	if err != nil {
		return nil, err
	}

	switch commandType {
	case '*':
		return p.parseArray(br)
	case '$':
		return p.parseBulkString(br)
	case ':':
		return p.parseInteger(br)
	case '-':
		return p.parseError(br)
	case '+':
		return p.parseSimpleString(br)
	case '_':
		return p.parseNull(br)
	default:
		return ErrorValue{"ERR unknown RESP type"}, nil
	}
}

func readLine(br *bufio.Reader) (string, error) {
	line, err := br.ReadString('\n')
	if err != nil {
		return "", err
	}
	lineLen := len(line)
	if lineLen < 2 || line[lineLen-2] != '\r' {
		return "", fmt.Errorf("protocol error: expected \\r\\n")
	}
	return line[:lineLen-2], nil
}

func (p *Parser) parseArray(br *bufio.Reader) (Value, error) {
	line, err := readLine(br)
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
		val, err := p.ReadValue(br)
		if err != nil {
			return nil, err
		}
		items = append(items, val)
	}
	return ArrayValue{Data: items}, nil
}

func (p *Parser) ReadRequest(r io.Reader) (*Request, error) {
	br := bufio.NewReader(r)
	val, err := p.ReadValue(br)
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

func (p *Parser) parseSimpleString(br *bufio.Reader) (Value, error) {
	line, err := readLine(br)
	if err != nil {
		return nil, fmt.Errorf("error parse SimpleStringValue: %w", err)
	}
	return SimpleStringValue{Data: []byte(line)}, nil
}

func (p *Parser) parseBulkString(br *bufio.Reader) (Value, error) {
	line, err := readLine(br)
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
	_, err = io.ReadFull(br, data)
	if err != nil {
		return nil, err
	}
	crlf := make([]byte, 2)
	_, err = io.ReadFull(br, crlf)
	if err != nil {
		return nil, err
	}
	if !(crlf[0] == '\r' && crlf[1] == '\n') {
		return nil, fmt.Errorf("protocol error: expected \\r\\n")
	}
	return BulkStringValue{Data: data}, nil
}

func (p *Parser) parseInteger(br *bufio.Reader) (Value, error) {
	line, err := readLine(br)
	if err != nil {
		return nil, fmt.Errorf("error parse IntegerValue: %w", err)
	}
	num, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error convert string to int: %w", err)
	}
	return IntegerValue{Value: num}, nil
}

func (p *Parser) parseError(br *bufio.Reader) (Value, error) {
	line, err := readLine(br)
	if err != nil {
		return nil, fmt.Errorf("error parse ErrorValue: %w", err)
	}

	return ErrorValue{Message: line}, nil
}

func (p *Parser) parseNull(br *bufio.Reader) (Value, error) {
	_, err := readLine(br)
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
