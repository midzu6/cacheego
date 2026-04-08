package store

import (
	"container/list"
)

type RedisValue interface {
	Type() string
	String() string
	Size() int64
}

// StringValue value in store map
type StringValue struct {
	Data []byte
}

func (sv StringValue) Type() string {
	return "string"
}

func (sv StringValue) String() string {
	return string(sv.Data)
}

func (sv StringValue) Size() int64 {
	return int64(len(sv.Data))
}

func (sv StringValue) Bytes() []byte {
	return sv.Data
}

// ListValue list in map
type ListValue struct {
	Data *list.List
}

func (lv ListValue) Type() string {
	return "list"
}

func (lv ListValue) String() string {
	return "[list]" // rewrite
}

func (lv ListValue) Size() int64 {
	return int64(lv.Data.Len())
}

func NewListValue() ListValue {
	return ListValue{
		Data: list.New(),
	}
}
