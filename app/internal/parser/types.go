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
	Data []byte
}

func (bsv BulkStringValue) Type() Type     { return TypeBulkString }
func (bsv BulkStringValue) Bytes() []byte  { return bsv.Data }
func (bsv BulkStringValue) String() string { return string(bsv.Data) }

// IntegerValue число :
type IntegerValue struct {
	Value int64
}

func (iv IntegerValue) Type() Type     { return TypeInteger }
func (iv IntegerValue) Bytes() []byte  { return nil }
func (iv IntegerValue) String() string { return strconv.FormatInt(iv.Value, 10) }

// ArrayValue массив *
type ArrayValue struct {
	Data []Value
}

func (av ArrayValue) Type() Type    { return TypeArray }
func (av ArrayValue) Bytes() []byte { return nil }
func (av ArrayValue) String() string {
	if len(av.Data) == 0 {
		return "[]"
	}
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, item := range av.Data {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(item.String())
	}
	sb.WriteString("]")
	return sb.String()
}
func (av ArrayValue) Elements() []Value {
	return av.Data
}

// SimpleStringValue простая строка +
type SimpleStringValue struct {
	Data []byte
}

func (ssv SimpleStringValue) Type() Type     { return TypeSimpleString }
func (ssv SimpleStringValue) Bytes() []byte  { return ssv.Data }
func (ssv SimpleStringValue) String() string { return string(ssv.Data) }

// ErrorValue ошибка -
type ErrorValue struct {
	Message string
}

func (ev ErrorValue) Type() Type     { return TypeError }
func (ev ErrorValue) Bytes() []byte  { return nil }
func (ev ErrorValue) String() string { return ev.Message }

// NullValue null _
type NullValue struct{}

func (nv NullValue) Type() Type     { return TypeNull }
func (nv NullValue) Bytes() []byte  { return nil }
func (nv NullValue) String() string { return "null" }
