package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

var _ = net.Listen
var _ = os.Exit

var (
	listen = flag.String("listen", ":6379", "address to listen to")
)

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

func main() {

	flag.Parse()

	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
