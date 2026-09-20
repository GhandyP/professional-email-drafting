package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GhandyP/professional-email-drafting/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cli.Run(ctx, os.Args[1:], cli.Deps{Now: time.Now}, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
