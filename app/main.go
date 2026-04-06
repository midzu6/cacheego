package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var _ = net.Listen
var _ = os.Exit

var (
	listen = flag.String("listen", ":6379", "address to listen to")
)

type Config struct {
	ListenAddr string
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

type Server struct {
	Config
	ln      net.Listener
	mu      sync.RWMutex
	storage map[string]entry
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = *listen
	}
	return &Server{
		Config:  cfg,
		storage: make(map[string]entry),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to bind to port %s", s.Config.ListenAddr)
	}
	defer closeIt(ln, &err, "close listener")
	slog.Info("server started", "addr", ln.Addr().String())

	s.ln = ln
	return s.acceptLoop()
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()

		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			slog.Error("accept failed", "error", err)
			continue
		}

		slog.Info("Client connected", "remote", conn.RemoteAddr().String())
		go s.handleConn(conn)
	}
	return nil
}

var bufPool = sync.Pool{New: func() interface{} {
	return make([]byte, 512)
}}

func (s *Server) handleConn(conn net.Conn) {
	var err error
	defer closeIt(conn, &err, "close connection")

	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)

	var leftover []byte

	for {
		n, err := conn.Read(buf)
		if errors.Is(err, io.EOF) {
			slog.Info("connection closed", "remote", conn.RemoteAddr())
			return
		}
		if err != nil {
			slog.Error("read error", "err", err)
			return
		}

		data := append(leftover, buf[:n]...)
		leftover = nil

		pos := 0
		for pos < len(data) {
			consumed, cmd, parseErr := parseArray(data[pos:])

			if errors.Is(parseErr, ErrIncompleteData) {
				leftover = make([]byte, len(data[pos:]))
				copy(leftover, data[pos:])
				break
			}
			if parseErr != nil {
				slog.Error("parse error", "err", parseErr)
				return
			}

			pos += consumed

			switch cmd.Name {
			case "PING":
				_, err = conn.Write([]byte("+PONG\r\n"))
			case "ECHO":
				if len(cmd.Args) == 0 {
					_, err = conn.Write([]byte("-ERR wrong number of arguments for 'echo' command\r\n"))
				} else {
					_, err = conn.Write(encodeBulkString(cmd.Args[0]))
				}
			case "SET":
				argLen := len(cmd.Args)
				if argLen < 2 {
					slog.Error("invalid args length", "length", len(cmd.Args))
					_, err = conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
					continue
				}

				if argLen > 2 {
					for i := 2; i < len(cmd.Args); i += 2 {
						if i+1 >= len(cmd.Args) {
							slog.Error("invalid args length", "length", len(cmd.Args))
							_, err = conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
							break
						}
						if strings.ToUpper(string(cmd.Args[i])) == "EX" {
							t, convErr := strconv.Atoi(string(cmd.Args[i+1]))
							if convErr != nil {
								slog.Error("invalid time", "err", err)
								_, err = conn.Write([]byte("-ERR wrong time parameter for 'set' command\r\n"))
								return
							}
							s.mu.Lock()
							s.storage[string(cmd.Args[0])] = entry{
								value:     cmd.Args[1],
								expiresAt: time.Now().Add(time.Duration(t) * time.Second),
							}
							s.mu.Unlock()
						} else if strings.ToUpper(string(cmd.Args[i])) == "PX" {
							t, convErr := strconv.Atoi(string(cmd.Args[i+1]))
							if convErr != nil {
								slog.Error("invalid time", "err", err)
								_, err = conn.Write([]byte("-ERR wrong time parameter for 'set' command\r\n"))
								return
							}
							s.mu.Lock()
							s.storage[string(cmd.Args[0])] = entry{
								value:     cmd.Args[1],
								expiresAt: time.Now().Add(time.Duration(t) * time.Millisecond),
							}
							s.mu.Unlock()
						}
					}
					_, err = conn.Write([]byte("+OK\r\n"))

				} else {
					s.mu.Lock()
					s.storage[string(cmd.Args[0])] = entry{
						value: cmd.Args[1],
					}
					s.mu.Unlock()
					_, err = conn.Write([]byte("+OK\r\n"))

				}

			case "GET":
				if len(cmd.Args) < 1 {
					slog.Error("invalid args length", "length", len(cmd.Args))
					_, err = conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
					continue
				}

				s.mu.RLock()
				entr, ok := s.storage[string(cmd.Args[0])]
				s.mu.RUnlock()

				if !ok {
					slog.Info("key not exists", "key", string(cmd.Args[0]))
					_, err = conn.Write([]byte("$-1\r\n"))
				} else if !entr.expiresAt.IsZero() && time.Now().After(entr.expiresAt) {
					slog.Info("key expired", "key", string(cmd.Args[0]))
					_, err = conn.Write([]byte("$-1\r\n"))
				} else {
					_, err = conn.Write(encodeBulkString(entr.value))
				}

			default:
				_, err = conn.Write([]byte("-ERR unknown command\r\n"))
			}

			if err != nil {
				slog.Error("write error", "err", err)
				return
			}
		}
	}
}

func run() (err error) {
	cfg := Config{
		ListenAddr: *listen,
	}
	srv := NewServer(cfg)

	err = srv.Start()
	if err != nil {
		return fmt.Errorf("cannot start server %w", err)
	}

	return nil
}

func closeIt(c io.Closer, errp *error, msg string) {
	if closeErr := c.Close(); closeErr != nil && *errp == nil {
		*errp = fmt.Errorf("%s: %w", msg, closeErr)
	} else if closeErr != nil {
		slog.Warn("failed to close resource after error",
			"resource", msg,
			"original_error", *errp,
			"close_error", closeErr)
	}

}

func main() {

	flag.Parse()

	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
