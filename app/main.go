package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codecrafters-io/redis-starter-go/app/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/app/internal/server"
	"github.com/codecrafters-io/redis-starter-go/app/internal/store"
)

var listen = flag.String("listen", ":6379", "address to listen to")

func main() {
	flag.Parse()

	// Создаём компоненты
	st := store.NewStore()

	reg := commands.NewRegistry()
	reg.Register(&commands.PingCommand{})
	reg.Register(&commands.EchoCommand{})
	reg.Register(&commands.SetCommand{})
	reg.Register(&commands.GetCommand{})
	reg.Register(&commands.DeleteCommand{})

	srv := server.New(server.Config{ListenAddr: *listen}, st, reg)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	fmt.Println("\nShutting down gracefully")
	srv.Close()
}
