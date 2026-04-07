package parser

import (
	"strconv"
	"strings"
)

const (
	TypeSimpleString Type = "simple"
	TypeInteger      Type = "integer"
	TypeBulkString   Type = "bulk"
	TypeArray        Type = "array"
	TypeError        Type = "error"
	TypeNull         Type = "null"
)

type Type string

type Value interface {
	Type() Type
	String() string
	Bytes() []byte
}

// BulkStringValue строка $
type BulkStringValue struct {
	data []byte
}

func (bsv BulkStringValue) Type() Type     { return TypeBulkString }
func (bsv BulkStringValue) Bytes() []byte  { return bsv.data }
func (bsv BulkStringValue) String() string { return string(bsv.data) }

// IntegerValue число :
type IntegerValue struct {
	value int64
}

func (iv IntegerValue) Type() Type     { return TypeInteger }
func (iv IntegerValue) Bytes() []byte  { return nil }
func (iv IntegerValue) String() string { return strconv.FormatInt(iv.value, 10) }

// ArrayValue массив *
type ArrayValue struct {
	data []Value
}

func (av ArrayValue) Type() Type     { return TypeArray }
func (av ArrayValue) Bytes() []Value { return nil }
func (av ArrayValue) String() string {
	if len(av.data) == 0 {
		return "[]"
	}
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, item := range av.data {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(item.String())
	}
	sb.WriteString("]")
	return sb.String()
}
func (av ArrayValue) Elements() []Value {
	return av.data
}

// SimpleStringValue простая строка +
type SimpleStringValue struct {
	data []byte
}

func (ssv SimpleStringValue) Type() Type     { return TypeSimpleString }
func (ssv SimpleStringValue) Bytes() []byte  { return ssv.data }
func (ssv SimpleStringValue) String() string { return string(ssv.data) }

// ErrorValue ошибка -
type ErrorValue struct {
	message string
}

func (ev ErrorValue) Type() Type     { return TypeError }
func (ev ErrorValue) Bytes() []byte  { return nil }
func (ev ErrorValue) String() string { return ev.message }

// NullValue null _
type NullValue struct{}

func (nv NullValue) Type() Type     { return TypeNull }
func (nv NullValue) Bytes() []byte  { return nil }
func (nv NullValue) String() string { return "null" }
