package layers

import "github.com/shimmeringbee/da"

type PassThruLayer struct{}

var _ OutputLayer = (*PassThruLayer)(nil)

func (p PassThruLayer) Name() string {
	return "PassThru"
}

func (p PassThruLayer) Capability(rl RetentionLevel, c da.Capability, d da.Device) interface{} {
	return d.Gateway().Capability(c)
}

func (p PassThruLayer) MaintainedStatus(c da.Capability, d da.Device) interface{} {
	return struct{}{}
}

var _ OutputStack = (*PassThruStack)(nil)

type PassThruStack struct {
	Layer PassThruLayer
}

func (p PassThruStack) Layers() []string {
	return []string{p.Layer.Name()}
}

func (p PassThruStack) Lookup(name string) OutputLayer {
	return p.Layer
}
