package layers

import "github.com/shimmeringbee/da"

type PassThruLayer struct{}

var _ OutputLayer = (*PassThruLayer)(nil)

func (p PassThruLayer) Name() string {
	return "PassThru"
}

func (p PassThruLayer) Device(rl RetentionLevel, d da.Device) da.Device {
	return d
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
