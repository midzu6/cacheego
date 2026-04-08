package store

type RedisValue interface {
	Type() string
	String() string
	Size() int64
	Bytes() []byte
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
