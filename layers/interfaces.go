package layers

import "github.com/shimmeringbee/da"

type RetentionLevel uint8

const (
	OneShot  RetentionLevel = 0
	Maintain RetentionLevel = 1
)

type OutputStack interface {
	Layers() []string
	Lookup(name string) OutputLayer
}

type OutputLayer interface {
	Name() string
	Capability(rl RetentionLevel, c da.Capability, d da.Device) interface{}
	MaintainedStatus(c da.Capability, d da.Device) interface{}
}
