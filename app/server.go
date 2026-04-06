package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
)

type Config struct {
	ListenAddr string
}

type Server struct {
	Config
	ln      net.Listener
	mu      sync.RWMutex
	storage map[string]Entry
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = *listen
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		Config:  cfg,
		storage: make(map[string]Entry),
		ctx:     ctx,
		cancel:  cancel,
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
	go s.cleanupExpired()

	return s.acceptLoop()
}

func (s *Server) Close() {
	s.cancel()
	err := s.ln.Close()
	if err != nil {
		return
	}
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
				res, ecErr := s.cmdEcho(cmd.Args)
				if ecErr != nil {
					_, err = conn.Write([]byte("-" + ecErr.Error() + "\r\n"))
				} else {
					_, err = conn.Write(encodeBulkString(res))
				}
			case "SET":
				if err = s.cmdSet(cmd.Args); err != nil {
					_, err = conn.Write([]byte("-" + err.Error() + "\r\n"))
				} else {
					_, err = conn.Write([]byte("+OK\r\n"))
				}
			case "GET":
				val, errGet := s.cmdGet(cmd.Args)

				if errors.Is(errGet, ErrKeyNotExists) {
					slog.Info("key not exists", "key", string(cmd.Args[0]))
					_, err = conn.Write([]byte("$-1\r\n"))
				} else if errors.Is(errGet, ErrKeyIsExpired) {
					slog.Info("key expired", "key", string(cmd.Args[0]))
					_, err = conn.Write([]byte("$-1\r\n"))
				} else if errors.Is(errGet, ErrWrongNumberOfArgument) {
					slog.Info("not enough args", "err", errGet)
					_, err = conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
				} else {
					_, err = conn.Write(encodeBulkString(val))
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
