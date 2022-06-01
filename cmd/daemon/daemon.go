package main

import (
	"context"
	"fmt"
	_ "github.com/gdamore/tcell/termbox"
	"github.com/jtcressy-home/edged/pkg/config"
	"github.com/jtcressy-home/edged/pkg/controller"
	_ "image/png"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	c := &config.Config{}

	if err := c.Init(os.Args); err != nil {
		log.Fatal(err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ctl, err := controller.NewController(c)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		ctl.CleanUp()
		signal.Stop(signalChan)
		cancel()
	}()

	go func() {
		for {
			select {
			case s := <-signalChan:
				switch s {
				case syscall.SIGINT, syscall.SIGTERM:
					log.Printf("got SIGINT/SIGTERM, exiting.")
					cancel()
					os.Exit(0)
				case syscall.SIGHUP:
					log.Printf("Got SIGHUP, reloading.")
					if err := c.Init(os.Args); err != nil {
						log.Fatal(err)
					}
				}
			case <-ctx.Done():
				log.Printf("Done.")
				os.Exit(0)
			}
		}
	}()

	if err := ctl.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

//TODO: on device startup or init:
// - Gather device information
// - Ensure hostname is derived from board serial numbers or identifiers
// - (Optionally) set hostname via tailscale local client
// - figure out display options such as oled, lcd or http kiosk server (local port)
//TODO: once tailscaled is at NeedsLogin stage:
// - Convert AuthURL to QR Code and display it
//TODO: after tailscaled is at Running stage:
// - Display device posture/info, such as
//    - Tailnet
//    - Tailscale IP/Hostname
//    - Serial Number
//    - Tailscale ACL Tags (should be updated as they change dynamically)
//    - Provisioning status (TBD)
//TODO: Implement poison-pill protocol:
// - Triggered by removing device from tailscale control admin interface (login.tailscale-utils.com)
// - Call Logout() and deprovision device
// - Delete kubelet/k3s agent config
// - remove containers/images
// - erase persistent storage
