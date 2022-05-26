package tailscale_utils

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"tailscale.com/client/tailscale"
	"tailscale.com/cmd/tailscale/cli"
	tsversion "tailscale.com/version"
)

var (
	wg        sync.WaitGroup
	goroCount = 0
)

type CliWrapper struct {
	isRunning bool
	mu        sync.Mutex
	StdOut    io.Writer
	StdErr    io.Writer
}

func (cw *CliWrapper) Run(args []string) error {
	if goroCount > 1 {
		return fmt.Errorf("only one running tailscale CLI execution allowed at a time")
	}
	cw.mu.Lock()
	start := !cw.isRunning
	cw.isRunning = true
	goroCount++
	cw.mu.Unlock()
	wg.Add(1)
	if start {
		go func() {
			defer wg.Done()
			err := cw.runCli(args)
			if err != nil {
				cw.StdErr.Write([]byte(fmt.Errorf("error running wrapped tailscale command: %v", err).Error()))
			}
			cw.mu.Lock()
			cw.isRunning = false
			goroCount--
			cw.mu.Unlock()
		}()
	}
	return nil
}

func (cw *CliWrapper) runCli(args []string) error {
	status, err := tailscale.Status(context.Background())
	if err != nil {
		return fmt.Errorf("err while getting tailscaled version: %v", err)
	}
	//TODO: This is _very_ hacky and should be replaced with an official api once it's implemented
	tsversion.Long = status.Version
	tsversion.GitDirty = false
	cli.Stdout = cw.StdOut
	cli.Stderr = cw.StdErr
	if err := cli.Run(args); err != nil {
		return fmt.Errorf("error running cli command with args '%v': %v", strings.Join(args, " "), err)
	}
	return nil
}

func (cw *CliWrapper) IsRunning() bool {
	return cw.isRunning
}

func (cw *CliWrapper) WaitForAll() {
	wg.Wait()
}
