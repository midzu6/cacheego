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

type pingCommand struct{}

func (pingCommand) Name() string { return "PING" }

func (pingCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	if len(args) == 0 {
		return parser.SimpleStringValue{Data: []byte("PONG")}, nil
	}
	return parser.BulkStringValue{Data: args[0].Bytes()}, nil
}

type echoCommand struct{}

func (echoCommand) Name() string { return "ECHO" }

func (echoCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	if len(args) == 0 {
		return nil, errors.New("ERR wrong number of arguments for 'echo' command")
	}
	return parser.BulkStringValue{Data: args[0].Bytes()}, nil
}

type setCommand struct{}

func (c *setCommand) Name() string { return "SET" }

func (c *setCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {

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

type getCommand struct{}

func (gc *getCommand) Name() string { return "GET" }

func (gc *getCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	key := args[0].String()
	val, ok := st.Get(key)

	if !ok {
		return parser.NullValue{}, nil
	}
	return parser.BulkStringValue{Data: val.Bytes()}, nil
}

type deleteCommand struct{}

func (dc *deleteCommand) Name() string { return "DEL" }

func (dc *deleteCommand) Execute(args []parser.Value, st store.Store) (parser.Value, error) {
	keys := make([]string, 0, len(args))
	for _, v := range args {
		keys = append(keys, v.String())
	}
	count := st.Delete(keys...)
	return parser.IntegerValue{Value: count}, nil

}
