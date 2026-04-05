package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

var (
	listen = flag.String("listen", ":6379", "address to listen to")
)

func run() (err error) {
	l, err := net.Listen("tcp", *listen)
	if err != nil {
		return fmt.Errorf("failed to bind to port %s", *listen)
	}
	defer closeIt(l, &err, "close listener")

	slog.Info("server started", "addr", l.Addr().String())

	conn, err := l.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept: %w", err)
	}
	defer closeIt(conn, &err, "close connection")

	slog.Info("client connected", "remote", conn.RemoteAddr().String())

	for {
		buf := make([]byte, 512)
		n, err := conn.Read(buf)

		if err == io.EOF {
			slog.Info("client disconnected", "remote", conn.RemoteAddr().String())
			break
		}
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		data := buf[:n]
		slog.Debug("received bytes", "count", n)

		countCommand := strings.Count(string(data), "PONG\r\n")
		for i := 0; i <= countCommand; i++ {
			_, err = conn.Write([]byte("+PONG\r\n"))
			if err != nil {
				return fmt.Errorf("failed to write response: %w", err)
			}
			slog.Debug("sent PONG", "iteration", i+1)
		}
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
