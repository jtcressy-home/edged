package oled

import (
	ui "github.com/gizak/termui/v3"
	"github.com/jtcressy-home/edged/pkg/display"
	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"image"
	"image/color"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ssd1306"
	"periph.io/x/host/v3"
	"sync"
)

//TODO

var (
	initonce sync.Once
)

type Oled struct {
	layout display.Layout
	dev    *ssd1306.Dev
	bus    i2c.BusCloser
	img    image.Image
}

func (o *Oled) Init() (err error) {
	if _, err := host.Init(); err != nil {
		return
	}

	// Open a handle to the first available I²C bus:
	o.bus, err = i2creg.Open("")
	if err != nil {
		return
	}

	// Open a handle to a ssd1306 connected on the I²C bus:
	o.dev, err = ssd1306.NewI2C(o.bus, &ssd1306.Opts{
		W:             128,
		H:             32,
		Rotated:       false,
		Sequential:    false,
		SwapTopBottom: false,
	})
	if err != nil {
		return
	}

	return
}

func (o *Oled) SetLayout(layout display.Layout) {
	o.layout = layout
}

func (o *Oled) PollEvents() <-chan ui.Event {
	//TODO implement me
	return make(chan ui.Event, 1) // useless channel
}

func (o *Oled) Refresh(data display.RefreshData) error {
	if data.TailscaleStatus.AuthURL != "" {
		q, err := qrcode.New(data.TailscaleStatus.AuthURL, qrcode.Medium)
		if err != nil {
			return err
		}
		q.DisableBorder = true
		o.img = q.Image(32)
	} else {
		o.img = 
		col := color.Gray{
			Y: 0,
		}
		d := &font.Drawer{
			Dst: o.img,
		}
	}
}

func (o *Oled) Render() {
	if o.img != nil {
		o.dev.Draw(o.img.Bounds(), o.img, image.Point{})
	}
}

func (o *Oled) Resize(width, height int) {
	return
}

func (o *Oled) Clear() {

}

func (o *Oled) CleanUp() {

}
