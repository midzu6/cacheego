package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrIncompleteData = fmt.Errorf("incomplete data")

type Command struct {
	Name string
	Args [][]byte
}

func findCRLF(b []byte) int {
	for i := 1; i < len(b); i++ {
		if b[i] == '\n' && b[i-1] == '\r' {
			return i + 1
		}
	}
	return -1
}

func parseBulkString(b []byte) (n int, data []byte, err error) {
	if len(b) == 0 || b[0] != '$' {
		return 0, nil, fmt.Errorf("expected '$' got %q", b[0])
	}
	crlfPos := findCRLF(b[1:])
	if crlfPos == -1 {
		return 0, nil, fmt.Errorf("incomplete bulk string header")
	}
	headerPosition := crlfPos + 1

	lengthBytes := b[1 : headerPosition-2]
	length, err := strconv.Atoi(string(lengthBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("invalid bulk length: %w", err)
	}
	if length == -1 {
		return headerPosition, nil, nil
	}

	neededLength := headerPosition + length + 2
	if neededLength > len(b) {
		return 0, nil, ErrIncompleteData
	}

	dataStart := headerPosition
	dataEnd := headerPosition + length

	if b[dataEnd] != '\r' || b[dataEnd+1] != '\n' {
		return 0, nil, fmt.Errorf("expected CRLF after bulk data")
	}

	data = b[dataStart:dataEnd]
	n = dataEnd + 2

	return n, data, nil
}

func parseArray(b []byte) (n int, cmd *Command, err error) {
	if len(b) == 0 || b[0] != '*' {
		return 0, nil, fmt.Errorf("expected '*', got %q", b[0])
	}
	crlfPos := findCRLF(b[1:])
	if crlfPos < 0 {
		return 0, nil, ErrIncompleteData
	}
	headerPos := crlfPos + 1
	count, err := strconv.Atoi(string(b[1 : headerPos-2]))
	if err != nil {
		return 0, nil, fmt.Errorf("invalid array length: %w", err)
	}
	if count == 0 {
		return headerPos, &Command{}, nil
	}

	args := make([][]byte, 0, count-1)
	var name string
	pos := headerPos
	for i := 0; i < count; i++ {
		if pos > len(b) {
			return 0, nil, ErrIncompleteData
		}

		if b[pos] != '$' {
			return 0, nil, fmt.Errorf("expected bulk string, got %q", b[pos])
		}

		ln, data, parseErr := parseBulkString(b[pos:])
		if parseErr != nil {
			if errors.Is(parseErr, ErrIncompleteData) {
				return 0, nil, ErrIncompleteData
			}
			return 0, nil, parseErr
		}
		if i == 0 {
			name = strings.ToUpper(string(data))
		} else {
			args = append(args, data)
		}
		pos += ln
	}
	return pos, &Command{Name: name, Args: args}, nil
}

func encodeBulkString(b []byte) []byte {
	capacity := 1 + len(strconv.Itoa(len(b))) + 2 + len(b) + 2
	resp := make([]byte, 0, capacity)
	resp = append(resp, '$')
	resp = strconv.AppendInt(resp, int64(len(b)), 10)
	resp = append(resp, '\r', '\n')
	resp = append(resp, b...)
	resp = append(resp, '\r', '\n')
	return resp
}
