package main

import (
	"context"
	"fmt"
	_ "github.com/gdamore/tcell/termbox"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/jtcressy-home/edged/pkg/config"
	tsutils "github.com/jtcressy-home/edged/pkg/tailscale_utils"
	"github.com/skip2/go-qrcode"
	_ "image/png"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"time"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	c := &config.Config{}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	defer func() {
		if c.EnableTUI {
			ui.Close()
		}
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
					if c.EnableTUI {
						ui.Close()
					}
					os.Exit(1)
				case syscall.SIGHUP:
					log.Printf("Got SIGHUP, reloading.")
					if err := c.Init(os.Args); err != nil {
						log.Fatal(err)
					}
				}
			case <-ctx.Done():
				log.Printf("Done.")
				if c.EnableTUI {
					ui.Close()
				}
				os.Exit(1)
			}
		}
	}()

	if err := run(ctx, c); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, c *config.Config) error {
	if err := c.Init(os.Args); err != nil {
		log.Fatal(err)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(c.Tick)
	defer ticker.Stop()

	cw := &tsutils.CliWrapper{
		StdOut: log.Writer(),
		StdErr: log.Writer(),
	}
	defer cw.WaitForAll()

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

loop:
	for {
		//Core functionality
		status, err := tailscale.Status(ctx)
		if err != nil {
			return err
		}

		// Global UI helpers
		termWidth, termHeight := ui.TerminalDimensions()
		grid := ui.NewGrid()
		grid.SetRect(0, 0, termWidth, termHeight)

		switch status.BackendState {
		case ipn.NeedsLogin.String():
			qrImage := widgets.NewList()
			qrImage.Title = "Tailscale Login"
			if status.AuthURL != "" {
				q, err := qrcode.New(status.AuthURL, qrcode.Medium)
				if err != nil {
					return err
				}
				q.DisableBorder = true
				qrImage.Rows = strings.Split(q.ToString(true), "\n")
				qrImage.PaddingRight = 0
				qrImage.SetRect(0, 0, 32, 18)
			} else {
				qrImage.Rows = []string{"Status: Waiting for Auth URL"}
				//TODO: This is _very_ hacky and should be replaced with an official api once it's implemented
				if !cw.IsRunning() {
					if err := cw.Run([]string{"up", "--json"}); err != nil {
						return err
					}
				}
			}

			statusTable := widgets.NewTable()
			statusTable.Title = "Tailscale Status"
			statusTable.Rows = [][]string{
				{"Status", status.BackendState},
				{"Healthy", func() string {
					if status.Self.Online && len(status.Health) < 1 {
						return "Yes"
					} else {
						return fmt.Sprintf("No: %v", strings.Join(status.Health, ", "))
					}
				}()},
				{"Auth URL", status.AuthURL},
			}
			statusTable.PaddingRight = 1
			statusTable.PaddingLeft = 1
			statusTable.PaddingTop = 0
			statusTable.PaddingBottom = 1
			statusTable.RowSeparator = false

			grid.Set(
				ui.NewCol(1.0/4,
					qrImage,
				),
				ui.NewCol(2.0/4,
					ui.NewRow(1.0/2,
						statusTable,
					),
				),
			)
			ui.Render(grid)
		case ipn.Running.String(), ipn.Stopped.String():
			statusTable := widgets.NewTable()
			statusTable.Title = "Tailscale Status"
			statusTable.Rows = [][]string{
				{"Status", status.BackendState},
				{"Healthy", func() string {
					if status.Self.Online && len(status.Health) < 1 {
						return "Yes"
					} else {
						return fmt.Sprintf("No: %v", strings.Join(status.Health, ", "))
					}
				}()},
				{"Current Tailnet", status.CurrentTailnet.Name},
				{"Hostname", status.Self.HostName},
				{"User Login", status.User[status.Self.UserID].LoginName},
				{"Device IP", func() string {
					if len(status.TailscaleIPs) > 0 {
						return status.TailscaleIPs[0].String()
					} else {
						return "<none>"
					}
				}()},
			}
			statusTable.PaddingRight = 1
			statusTable.PaddingLeft = 1
			statusTable.PaddingTop = 0
			statusTable.PaddingBottom = 1
			statusTable.RowSeparator = false

			ls := widgets.NewList()
			ls.Title = "Tailscale IPs"
			ls.Rows = []string{}
			for _, ip := range status.TailscaleIPs {
				ls.Rows = append(ls.Rows, ip.String())
			}

			grid.Set(
				ui.NewRow(1.0/2,
					ui.NewRow(1,
						ui.NewCol(1.0/2, statusTable),
						ui.NewCol(1.0/2, ls),
					),
				),
				//ui.NewRow(1.0/2,
				//	ui.NewCol(1.0/2, ls),
				//),
			)
			ui.Render(grid)
		default:
			log.Default().Printf("Unknown state: %v", status.BackendState)
		}

		// Rate/Control code
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			continue
		case <-interrupt:
			break loop
		case e := <-ui.PollEvents():
			switch e.ID {
			case "q", "<C-c>":
				log.Default().Println("Received quit command from TUI")
				return syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			case "l":
				if status.BackendState == ipn.Running.String() {
					if err := tailscale.Logout(context.Background()); err != nil {
						log.Fatalf("error running logout command: %v", err)
						return err
					}
				}
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			}
		}
	}
	return nil
}
