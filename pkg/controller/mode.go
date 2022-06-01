package controller

type Mode int

const (
	Bootstrap = Mode(iota)
	Provisioning
	ConfigurationPending
	Running
)

func (m Mode) String() string {
	return [...]string{
		"Bootstrap",
		"Provisioning",
		"ConfigurationPending",
		"Running",
	}[m]
}
