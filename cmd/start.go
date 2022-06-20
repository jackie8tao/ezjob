package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackie8tao/ezjob/ezjob"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use: "start",
	Run: startHandler,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func startHandler(_ *cobra.Command, _ []string) {
	var err error
	defer func() {
		if err != nil {
			log.Fatalf("ezjob start error: %v", err)
		}
	}()

	cfg, err := ezjob.LoadCfg(cfgFile)
	if err != nil {
		return
	}

	server, err := ezjob.NewEzJob(cfg)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	err = server.Serve(ctx)
	if err != nil {
		cancel()
		return
	}

	stopped := make(chan struct{}, 1)

	select {
	case <-sig:

		go func() {
			cancel()
			_ = server.Close()
			stopped <- struct{}{}
		}()

		select {
		case <-stopped:
		case <-time.After(3 * time.Second):
		}
	}
}
