package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/joho/godotenv"

	"go-lamp.autonomous.ai/bootstrap"
	"go-lamp.autonomous.ai/bootstrap/config"
	"go-lamp.autonomous.ai/lib/logger"
)

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(config.BootstrapVersion)
		return
	}

	// Load shared env file before logger init (so GELF_* env vars are visible).
	// Missing file is non-fatal — env may also be supplied by systemd.
	_ = godotenv.Load("/opt/lelamp/.env")

	cleanup := logger.Init(slog.LevelDebug, "/var/log/lumi-bootstrap.log")
	defer cleanup()

	b, err := bootstrap.ProvideServer()
	if err != nil {
		log.Fatalf("bootstrap: initialize: %v", err)
	}
	if err := b.Serve(); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}
	slog.Info("bootstrap stopped", "component", "bootstrap")
}
