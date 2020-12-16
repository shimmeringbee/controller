package main

import "github.com/shimmeringbee/da"

type RetentionLevel uint8

const (
	OneShot  RetentionLevel = 0
	Maintain RetentionLevel = 1
)

type OutputStack interface {
	Lookup(name string) OutputLayer
}

type OutputLayer interface {
	Name() string
	Capability(rl RetentionLevel, c da.Capability, d da.Device) interface{}
	MaintainedStatus(c da.Capability, d da.Device) interface{}
}

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

type PassThruStack struct{}

func (p PassThruStack) Lookup(name string) OutputLayer {
	return PassThruLayer{}
}
