package config

import (
	"github.com/namsral/flag"
	"io"
	"strings"
	"tailscale.com/client/tailscale"
	tspaths "tailscale.com/paths"
	"time"
)

const defaultTick = 60 * time.Second

type Config struct {
	Tick         time.Duration
	DisplayTypes []string
	LogOutput    io.Writer
}

func (c *Config) Init(args []string) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.String(flag.DefaultConfigFlagname, "", "Path to config file")

	var (
		tailscaledSocket = flags.String("socket", tspaths.DefaultTailscaledSocket(), "Path to tailscaled's unix socket")
		displayTypes     = flags.String("displays", "oled", "Display types: any of lcd,oled,tui")
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
	c.DisplayTypes = strings.Split(*displayTypes, ",")

	return nil
}
