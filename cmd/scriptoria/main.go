package main

import (
	"log/slog"
	"os"

	"github.com/KyleBrandon/scriptoria/pkg/server"
)

func main() {
	err := server.InitializeServer()
	if err != nil {
		slog.Error("failed to initialize the server")
		os.Exit(1)
	}
}
