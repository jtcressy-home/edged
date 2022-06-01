package controller

import (
	"context"
	ui "github.com/gizak/termui/v3"
	"github.com/jtcressy-home/edged/pkg/config"
	"github.com/jtcressy-home/edged/pkg/display"
	tsutils "github.com/jtcressy-home/edged/pkg/tailscale_utils"
	"log"
	"os"
	"os/signal"
	"syscall"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"time"
)

type Controller struct {
	c             *config.Config
	d             *display.Set
	Mode          Mode
	ticker        *time.Ticker
	cw            *tsutils.CliWrapper
	interruptChan chan os.Signal
}

func (c *Controller) Run(ctx context.Context) error {
	log.Println("Starting edged controller")
	defer c.cw.WaitForAll()
	defer c.ticker.Stop()
	signal.Notify(c.interruptChan, os.Interrupt)
loop:
	for {
		tailscaleStatus, err := tailscale.Status(ctx)
		if err != nil {
			return err
		}
		if tailscaleStatus.BackendState == ipn.NeedsLogin.String() && tailscaleStatus.AuthURL == "" {
			//Trigger interactive login
			//TODO: This is VERY hacky and should be replaced with an official api once it's implemented
			if !c.cw.IsRunning() {
				if err := c.cw.Run([]string{"up"}); err != nil {
					return err
				}
			}
		}

		switch tailscaleStatus.BackendState {
		case ipn.Running.String():
			if c.Mode == Bootstrap {
				c.Mode = Running
			}
		case ipn.NeedsLogin.String():
			if c.Mode != Bootstrap {
				c.Mode = Bootstrap
			}
		}

		//TODO: Figure out what needs to trigger Provisioning mode

		if c.Mode == Bootstrap {
			c.d.SetLayout(display.Bootstrap)
		} else if c.Mode == ConfigurationPending {
			c.d.SetLayout(display.Configuration)
		} else {
			c.d.SetLayout(display.Running)
		}

		//Refresh all status info and send to displays
		if err := c.d.Refresh(display.RefreshData{
			TailscaleStatus: tailscaleStatus,
		}); err != nil {
			return err
		}

		//Render displays
		c.d.Clear()
		c.d.Render()

		//Handle end of loop
		select {
		case <-ctx.Done():
			return nil
		case <-c.ticker.C:
			continue
		case <-c.interruptChan:
			break loop
		case e := <-c.d.PollEvents():
			switch e.ID {
			case "q", "<C-c>":
				log.Default().Println("Received quit command from TUI")
				return syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			case "<L-l>": //Logout
				if c.Mode == Running {
					if err := tailscale.Logout(ctx); err != nil {
						log.Fatalf("error running logout command: %v", err)
						return err
					}
				}
			case "<F2>": //Configure
				c.Mode = ConfigurationPending
				c.d.SetLayout(display.Configuration)
			case "<Resize>": //TODO: Capture this inside display.go instead somehow
				payload := e.Payload.(ui.Resize)
				c.d.Resize(payload.Width, payload.Height)
				c.d.Clear()
				c.d.Render()
			}
		}
	}
	return nil
}

func (c *Controller) CleanUp() {
	c.d.CleanUp()
}

func NewController(c *config.Config) (*Controller, error) {
	displays, err := func() (d []display.Display, err error) {
		for _, dt := range c.DisplayTypes {
			if dt == "tui" {
				d = append(d, &display.Tui{})
			}
			//TODO:
			//if dt == "oled" {
			//	d = append(d, &display.Oled{})
			//}
			//if dt == "lcd" {
			//	d = append(d, &display.Lcd{})
			//}
		}
		return
	}()
	if err != nil {
		return nil, err
	}
	d, err := display.NewSet(displays...)
	if err != nil {
		return nil, err
	}
	ctl := &Controller{
		c:      c,
		d:      d,
		Mode:   Bootstrap,
		ticker: time.NewTicker(c.Tick),
		cw: &tsutils.CliWrapper{
			StdErr: log.Writer(),
			StdOut: log.Writer(),
		},
		interruptChan: make(chan os.Signal, 1),
	}
	return ctl, nil
}
