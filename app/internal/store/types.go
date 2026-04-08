package store

type RedisValue interface {
	Name() string
	String() string
	Size() int64
}

// StringValue value in store map
type StringValue struct {
	data []byte
}

func (sv *StringValue) Name() string {
	return "string"
}

func (sv *StringValue) String() string {
	return string(sv.data)
}

func (sv *StringValue) Size() int64 {
	return int64(len(sv.data))
}
