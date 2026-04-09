package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/internal/parser"
	"github.com/codecrafters-io/redis-starter-go/app/internal/store"
)

type PingCommand struct{}

func (PingCommand) Name() string { return "PING" }

func (PingCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	if len(args) == 0 {
		return parser.SimpleStringValue{Data: []byte("PONG")}, nil
	}
	return parser.BulkStringValue{Data: args[0].Bytes()}, nil
}

type EchoCommand struct{}

func (EchoCommand) Name() string { return "ECHO" }

func (EchoCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	if len(args) == 0 {
		return nil, errors.New("ERR wrong number of arguments for 'echo' command")
	}
	return parser.BulkStringValue{Data: args[0].Bytes()}, nil
}

type SetCommand struct{}

func (c *SetCommand) Name() string { return "SET" }

func (c *SetCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {

	if len(args) < 2 {
		return nil, errors.New("ERR wrong number of arguments for 'set' command")
	}
	key := args[0].String()
	valueData := args[1].Bytes()
	value := store.StringValue{Data: valueData}

	var ttl time.Duration = 0

	for i := 2; i < len(args); i += 2 {
		if (i + 1) >= len(args) {
			return nil, errors.New("ERR syntax error")
		}

		opt := strings.ToUpper(args[i].String())
		durationStr := args[i+1].String()
		tm, err := strconv.ParseInt(durationStr, 10, 64)
		if err != nil || tm <= 0 {
			return nil, errors.New("ERR invalid expire time")
		}
		switch opt {
		case "EX":
			ttl = time.Duration(tm) * time.Second
		case "PX":
			ttl = time.Duration(tm) * time.Millisecond
		default:
			return nil, fmt.Errorf("ERR unknown option '%s'", opt)

		}
	}
	st.Set(key, value, ttl)
	return parser.SimpleStringValue{Data: []byte("OK")}, nil
}

type GetCommand struct{}

func (gc *GetCommand) Name() string { return "GET" }

func (gc *GetCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	key := args[0].String()
	val, ok := st.Get(key)

	if !ok {
		return parser.NullValue{}, nil
	}
	return parser.BulkStringValue{Data: val.(store.StringValue).Bytes()}, nil
}

type DeleteCommand struct{}

func (dc *DeleteCommand) Name() string { return "DEL" }

func (dc *DeleteCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	keys := make([]string, 0, len(args))
	for _, v := range args {
		keys = append(keys, v.String())
	}
	count := st.Delete(keys...)
	return parser.IntegerValue{Value: count}, nil

}

type RpushCommand struct{}

func (rp *RpushCommand) Name() string { return "RPUSH" }

func (rp *RpushCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {

	if len(args) < 2 {
		return nil, errors.New("ERR wrong number of arguments for 'RPUSH' command")
	}
	key := args[0].String()
	values := make([]string, 0, len(args)-1)
	for _, arg := range args[1:] {
		values = append(values, arg.String())
	}
	newLen, err := st.RPush(key, values...)
	if err != nil {
		return nil, err
	}

	return parser.IntegerValue{Value: newLen}, nil
}

type LRangeCommand struct{}

func (lrc *LRangeCommand) Name() string {
	return "LRANGE"
}

func (lrc *LRangeCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	if len(args) < 3 {
		return nil, errors.New("ERR wrong number of arguments for 'LRANGE' command")
	}
	key := args[0].String()
	startStr := args[1].String()
	stopStr := args[2].String()
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return nil, errors.New("ERR value is not an integer or out of range")
	}

	stop, err := strconv.Atoi(stopStr)
	if err != nil {
		return nil, errors.New("ERR value is not an integer or out of range")
	}

	arr, err := st.LRange(key, start, stop)
	if err != nil {
		return nil, err
	}
	data := make([]parser.Value, 0, len(arr))
	for _, v := range arr {
		val := parser.BulkStringValue{Data: []byte(v)}
		data = append(data, val)
	}

	return parser.ArrayValue{Data: data}, nil

}
