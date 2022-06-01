package display

import (
	ui "github.com/gizak/termui/v3"
	"reflect"
)

type Layout int

const (
	Bootstrap = Layout(iota)
	Running
	Configuration
)

type Display interface {
	Init() error
	SetLayout(layout Layout)
	PollEvents() <-chan ui.Event
	Refresh(data RefreshData) error
	Render()
	Resize(width, height int)
	Clear()
	CleanUp()
}

type Set struct {
	Display
	displays map[reflect.Type]Display
}

func (ds *Set) Init() (err error) {
	for _, d := range ds.displays {
		err = d.Init()
		if err != nil {
			return err
		}
	}
	return
}

func (ds *Set) PollEvents() <-chan ui.Event {
	//TODO: Expand this wrapper to inject hardware-based events, like a directional joystick on a RPi HAT
	return ui.PollEvents()
}

func (ds *Set) SetLayout(layout Layout) {
	for _, d := range ds.displays {
		d.SetLayout(layout)
	}
}

func (ds *Set) Refresh(data RefreshData) (err error) {
	for _, d := range ds.displays {
		err = d.Refresh(data)
		if err != nil {
			return
		}
	}
	return
}

func (ds *Set) Render() {
	for _, d := range ds.displays {
		d.Render()
	}
}

func (ds *Set) Resize(width int, height int) {
	for _, d := range ds.displays {
		d.Resize(width, height)
	}
}

func (ds *Set) Clear() {
	for _, d := range ds.displays {
		d.Clear()
	}
}

func (ds *Set) CleanUp() {
	for _, d := range ds.displays {
		d.CleanUp()
	}
}

func NewSet(displays ...Display) (*Set, error) {
	ds := &Set{}
	ds.displays = map[reflect.Type]Display{}
	for _, d := range displays {
		ds.displays[reflect.TypeOf(d)] = d
	}
	err := ds.Init()
	if err != nil {
		return nil, err
	}
	return ds, nil
}
