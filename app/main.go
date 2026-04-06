package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
)

var _ = net.Listen
var _ = os.Exit

var (
	listen = flag.String("listen", ":6379", "address to listen to")
)

type Config struct {
	ListenAddr string
}

type Server struct {
	Config
	ln net.Listener
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = *listen
	}
	return &Server{
		Config: cfg,
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
	var n int
	defer closeIt(conn, &err, "close connection")

	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)
OuterLoop:
	for {
		n, err = conn.Read(buf)
		if errors.Is(err, io.EOF) {
			slog.Info("connection closed")
			break
		}
		if err != nil {
			slog.Error("read error", "err", err)
			break
		}
		data := buf[:n]
		slog.Debug("received bytes", "count", n)
		countCmd := strings.Count(string(data), "PING\r\n")

		for range countCmd {
			_, err = conn.Write([]byte("+PONG\r\n"))

			if err != nil {
				slog.Error("write error", "err", err)
				break OuterLoop
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
