package exporter

import (
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
)

func ExportGateway(gw da.Gateway) ExportedGateway {
	caps := []string{}

	for _, c := range gw.Capabilities() {
		caps = append(caps, capabilities.StandardNames[c])
	}

	return ExportedGateway{
		Capabilities: caps,
		SelfDevice:   gw.Self().Identifier().String(),
	}
}
