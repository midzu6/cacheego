package parser

import (
	"bufio"
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
		return parseBulkString(p.r)
	case ':':
		return parseInteger(p.r)
	case '-':
		return parseError(p.r)
	case '+':
		return parseSimpleString(p.r)
	case '_':
		return parseNull(p.r)
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
		return nil, fmt.Errorf("error parse ArrayValue")
	}
	count, convErr := strconv.Atoi(line)
	if convErr != nil {
		return nil, fmt.Errorf("error convert string to int")
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
	return ArrayValue{data: items}, nil
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
	if len(arr.data) == 0 {
		return &Request{}, nil
	}

	nameValue := arr.data[0]
	name := strings.ToUpper(strings.TrimSpace(nameValue.String()))
	args := arr.data[1:]

	return &Request{
		Name: name,
		Args: args,
	}, nil
}

func parseSimpleString(r *bufio.Reader) (Value, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, fmt.Errorf("error parse SimpleStringValue")
	}
	return SimpleStringValue{data: []byte(line)}, nil
}

func parseBulkString(r *bufio.Reader) (Value, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, fmt.Errorf("error parse BulkStringValue")
	}
	n, err := strconv.Atoi(line)
	if err != nil {
		return nil, fmt.Errorf("error convert string to int")
	}
	data := make([]byte, n)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}
	crlf := make([]byte, 2)
	_, err = io.ReadFull(r, crlf)
	if err != nil {
		return nil, err
	}
	if !(crlf[0] == '\r' && crlf[1] == '\n') {
		return nil, fmt.Errorf("protocol error: expected \\r\\n")
	}
	return BulkStringValue{data: data}, nil
}

func parseInteger(r *bufio.Reader) (Value, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, fmt.Errorf("error parse IntegerValue")
	}
	num, err := strconv.Atoi(line)
	if err != nil {
		return nil, fmt.Errorf("error convert string to int")
	}
	return IntegerValue{value: int64(num)}, nil
}

func parseError(r *bufio.Reader) (Value, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, fmt.Errorf("error parse ErrorValue")
	}

	return ErrorValue{message: line}, nil
}

func parseNull(r *bufio.Reader) (Value, error) {
	_, err := readLine(r)
	if err != nil {
		return nil, fmt.Errorf("error parse NullValue")
	}
	return NullValue{}, nil
}
