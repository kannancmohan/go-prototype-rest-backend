package app_common

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type AppStopChan chan struct{}

type StopChan chan struct{}

// var _ StopChanReceiveOnly = StopChan(nil) //manually ensuring that StopChan implements StopChanReceiveOnly

// type StopChanReceiveOnly interface {
// 	Receive() <-chan struct{}
// }

// func (ch StopChan) Receive() <-chan struct{} {
// 	return ch
// }

func WaitForStopChan(ctx context.Context, stopChannels []StopChan) <-chan struct{} {
	shutdownChan := make(chan struct{})
	var once sync.Once // Ensure shutdownChan is closed only once

	// Start a goroutine for each stopChannel
	for _, ch := range stopChannels {
		go func(c StopChan) {
			select {
			case <-c: // Wait for the channel to be triggered
				slog.Debug("received stop signal in WaitForStopChan")
				once.Do(func() { close(shutdownChan) }) // Close shutdownChan once
			case <-ctx.Done(): // Handle context cancellation
				slog.Debug("received ctx cancel in WaitForStopChan")
				once.Do(func() { close(shutdownChan) }) // Close shutdownChan once
			}
		}(ch)
	}
	return shutdownChan
}

func SysInterruptStopChan() StopChan {
	stopChan := make(StopChan)

	sysInterruptChan := make(chan os.Signal, 1)
	signal.Notify(sysInterruptChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sysInterruptChan // block until sys interrupt signal
		slog.Debug("system shutdown signal received", "signal", sig.String())
		//stopChan <- struct{}{}
		close(stopChan)
	}()

	return stopChan
}
