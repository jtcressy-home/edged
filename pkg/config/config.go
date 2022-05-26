package config

import (
	ui "github.com/gizak/termui/v3"
	"github.com/namsral/flag"
	"io"
	"log"
	"os"
	"tailscale.com/client/tailscale"
	tspaths "tailscale.com/paths"
	"time"
)

const defaultTick = 60 * time.Second

type Config struct {
	Tick        time.Duration
	EnableTUI   bool
	DisplayType string
	LogOutput   io.Writer
}

func (c *Config) Init(args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.String(flag.DefaultConfigFlagname, "", "Path to config file")

	var (
		tailscaledSocket = flags.String("socket", tspaths.DefaultTailscaledSocket(), "Path to tailscaled's unix socket")
		displayType      = flags.String("display", "oled", "External display type: one of lcd,oled")
		termUI           = flags.Bool("tui", false, "Whether to use terminal UI or not - redirects log output to file")
		tick             = flags.Duration("tick", defaultTick, "Refresh interval on main loop")
	)

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	if tailscale.TailscaledSocket != *tailscaledSocket {
		tailscale.TailscaledSocket = *tailscaledSocket
		tailscale.TailscaledSocketSetExplicitly = true
	}

	c.Tick = *tick
	c.EnableTUI = *termUI
	c.DisplayType = *displayType

	//TODO: once we have a wrapper display package, call its init instead
	if c.EnableTUI {
		log.SetOutput(os.Stderr)
		if err := ui.Init(); err != nil {
			return err
		}
	} else {
		log.SetOutput(os.Stdout)
	}

	return nil
}
