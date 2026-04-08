package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/app/internal/parser"
	"github.com/codecrafters-io/redis-starter-go/app/internal/store"
)

type Config struct {
	ListenAddr string
}

type Server struct {
	Config
	ln       net.Listener
	store    store.Store
	registry *commands.Registry
	parser   *parser.Parser
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(cfg Config, st store.Store, reg *commands.Registry) *Server {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":6379"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		Config:   cfg,
		store:    st,
		registry: reg,
		parser:   parser.NewParser(nil),
		ctx:      ctx,
		cancel:   cancel,
	}
}
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to bind to port %s", s.Config.ListenAddr)
	}
	s.ln = ln
	slog.Info("server started", "addr", ln.Addr().String())

	go s.store.StartExpiry(s.ctx)

	return s.acceptLoop()
}

func (s *Server) Close() {
	s.cancel()
	if s.ln != nil {
		s.ln.Close()
	}
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			slog.Error("accept failed", "error", err)
			continue
		}

		slog.Info("Client connected", "remote", conn.RemoteAddr().String())
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	for {
		req, err := s.parser.ReadRequest()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Info("connection closed", "remote", conn.RemoteAddr())
				return
			}
			slog.Error("read/parse error", "err", err)
			return
		}
		cmd, ok := s.registry.Get(req.Name)
		if !ok {
			conn.Write([]byte("-ERR unknown command\r\n"))
			continue
		}

		resq, exErr := cmd.Execute(req.Args, s.store)
		if exErr != nil {
			conn.Write([]byte("-" + exErr.Error() + "\r\n"))
			continue
		}

		data, enErr := s.parser.Encode(resq)
		if enErr != nil {
			conn.Write([]byte("-" + enErr.Error() + "\r\n"))
			continue
		}
		conn.Write(data)
	}
}
