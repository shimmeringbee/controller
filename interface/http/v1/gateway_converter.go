package v1

import "github.com/shimmeringbee/da"

func convertDAGatewayToGateway(gw da.Gateway) ExportedGateway {
	capabilities := []string{}

	for _, cap := range gw.Capabilities() {
		uncastCapability := gw.Capability(cap)

		if basicCapability, ok := uncastCapability.(da.BasicCapability); ok {
			capabilities = append(capabilities, basicCapability.Name())
		}
	}

	return ExportedGateway{
		Capabilities: capabilities,
		SelfDevice:   gw.Self().Identifier().String(),
	}
}
