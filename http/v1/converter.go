package v1

import "github.com/shimmeringbee/da"

func convertDADeviceToDevice(daDevice da.Device) device {
	capabilities := map[string]interface{}{}

	for _, capFlag := range daDevice.Capabilities() {
		uncastCapability := daDevice.Gateway().Capability(capFlag)

		if basicCapability, ok := uncastCapability.(da.BasicCapability); ok {
			capabilities[basicCapability.Name()] = struct{}{}
		}
	}

	return device{
		Identifier:   daDevice.Identifier().String(),
		Capabilities: capabilities,
	}
}

func convertDAGatewayToGateway(gw da.Gateway) gateway {
	capabilities := []string{}

	for _, cap := range gw.Capabilities() {
		uncastCapability := gw.Capability(cap)

		if basicCapability, ok := uncastCapability.(da.BasicCapability); ok {
			capabilities = append(capabilities, basicCapability.Name())
		}
	}

	return gateway{
		Capabilities: capabilities,
		SelfDevice:   gw.Self().Identifier().String(),
	}
}
