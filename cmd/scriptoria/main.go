package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/KyleBrandon/scriptoria/pkg/server"
)

func main() {
	flag.Parse()
	err := server.InitializeServer()
	if err != nil {
		slog.Error("failed to initialize the server")
		os.Exit(1)
	}
}
