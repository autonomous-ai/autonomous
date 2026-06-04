package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/joho/godotenv"

	"go-lamp.autonomous.ai/lib/logger"
	"go-lamp.autonomous.ai/server"
	"go-lamp.autonomous.ai/server/config"
)

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(config.LampVersion)
		return
	}

	// Load shared env file before logger init (so GELF_* env vars are visible).
	// Missing file is non-fatal — env may also be supplied by systemd.
	_ = godotenv.Load("/opt/lelamp/.env")

	cleanup := logger.Init(slog.LevelDebug, "/var/log/lamp.log")
	defer cleanup()

	srv, err := server.InitializeServer()
	if err != nil {
		log.Fatal("initialize server: ", err)
	}
	if err := srv.Serve(func() {}); err != nil {
		log.Fatal("http server: ", err)
	}
}
