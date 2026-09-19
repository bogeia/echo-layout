package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bogeia/echo-layout/internal/config"
	"github.com/bogeia/echo-layout/internal/infra/logging"
	"github.com/bogeia/echo-layout/internal/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) (err error) {
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	path := flags.String("conf", "", "required YAML configuration file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(*path) == "" || flags.NArg() != 0 {
		return fmt.Errorf("usage: server --conf <config.yaml>")
	}
	config, err := config.Load(*path)
	if err != nil {
		return err
	}
	logger, closeLog, err := logging.New(config.Log)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeLog()) }()
	slog.SetDefault(logger)
	router, cleanup, err := wireRouter(logger)
	if err != nil {
		return err
	}
	defer cleanup()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.Run(ctx, router, config.Server)
}
