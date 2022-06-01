package display

import "tailscale.com/ipn/ipnstate"

type RefreshData struct {
	TailscaleStatus *ipnstate.Status
}
