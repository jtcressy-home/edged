package display

import (
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/skip2/go-qrcode"
	"strings"
	"sync"
)

var initonce sync.Once

type Tui struct {
	layout Layout
	output []ui.Drawable
}

func (d *Tui) CleanUp() {
	ui.Close()
}

func (d *Tui) Init() (err error) {
	initonce.Do(func() {
		err = ui.Init()
	})
	termWidth, termHeight := ui.TerminalDimensions()
	grid := ui.NewGrid()
	grid.SetRect(0, 0, termWidth, termHeight)
	d.output = append(d.output, grid)
	return err
}

func (d *Tui) SetLayout(layout Layout) {
	d.layout = layout
}

func (d *Tui) PollEvents() <-chan ui.Event {
	return ui.PollEvents()
}

func (d *Tui) Refresh(data RefreshData) (err error) {
	switch d.layout {
	case Bootstrap:
		qrImage := widgets.NewList()
		qrImage.Title = "Tailscale Login"
		if data.TailscaleStatus.AuthURL != "" {
			q, err := qrcode.New(data.TailscaleStatus.AuthURL, qrcode.Medium)
			if err != nil {
				return err
			}
			q.DisableBorder = true
			qrImage.Rows = strings.Split(q.ToString(true), "\n")
			qrImage.PaddingRight = 0
			qrImage.PaddingLeft = 1
			qrImage.SetRect(0, 0, len(qrImage.Rows[0])/2-6, len(qrImage.Rows)+2)
		} else {
			qrImage.Rows = []string{"Status: Waiting for Auth URL"}
			qrImage.SetRect(0, 0, len(qrImage.Rows[0])+4, len(qrImage.Rows)+2)
		}

		statusTable := widgets.NewTable()
		statusTable.Title = "Tailscale Status"
		statusTable.Rows = [][]string{
			{"Status", data.TailscaleStatus.BackendState},
			{"Healthy", func() string {
				if data.TailscaleStatus.Self.Online && len(data.TailscaleStatus.Health) < 1 {
					return "Yes"
				} else {
					return fmt.Sprintf("No: %v", strings.Join(data.TailscaleStatus.Health, ", "))
				}
			}()},
			{"Auth URL", data.TailscaleStatus.AuthURL},
		}
		maxRowLabelWidth := 0
		maxRowValueWidth := 0
		for _, r := range statusTable.Rows {
			if len(r[0]) > maxRowLabelWidth {
				maxRowLabelWidth = len(r[0])
			}
			if len(r[1]) > maxRowValueWidth {
				maxRowValueWidth = len(r[1])
			}
		}
		statusTable.PaddingRight = 1
		statusTable.PaddingLeft = 1
		statusTable.PaddingTop = 0
		statusTable.PaddingBottom = 1
		statusTable.RowSeparator = false
		statusTable.ColumnWidths = []int{maxRowLabelWidth, maxRowValueWidth}
		statusTable.FillRow = true
		statusTable.SetRect(qrImage.GetRect().Max.X, 0, qrImage.GetRect().Max.X*2, len(statusTable.Rows)+3)

		d.output = append(d.output, qrImage, statusTable)
	case Running:
		statusTable := widgets.NewTable()
		statusTable.Title = "Tailscale Status"
		statusTable.Rows = [][]string{
			{"Status", data.TailscaleStatus.BackendState},
			{"Healthy", func() string {
				if data.TailscaleStatus.Self.Online && len(data.TailscaleStatus.Health) < 1 {
					return "Yes"
				} else {
					return fmt.Sprintf("No: %v", strings.Join(data.TailscaleStatus.Health, ", "))
				}
			}()},
			{"Current Tailnet", data.TailscaleStatus.CurrentTailnet.Name},
			{"Hostname", data.TailscaleStatus.Self.HostName},
			{"User Login", data.TailscaleStatus.User[data.TailscaleStatus.Self.UserID].LoginName},
			{"Device IP", func() string {
				if len(data.TailscaleStatus.TailscaleIPs) > 0 {
					return data.TailscaleStatus.TailscaleIPs[0].String()
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

		d.output = append(d.output, statusTable)
	case Configuration:
		text := widgets.NewParagraph()
		text.Title = "Not Implemented"
		d.output = append(d.output, text)
	}
	return
}

func (d *Tui) Render() {
	ui.Render(d.output...)
}

func (d *Tui) Resize(width, height int) {
	for _, o := range d.output {
		o.SetRect(0, 0, width, height)
	}
}

func (d *Tui) Clear() {
	ui.Clear()
}
