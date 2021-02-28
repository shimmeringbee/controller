package v1

import "github.com/shimmeringbee/controller/metadata"

type device struct {
	Metadata     metadata.DeviceMetadata
	Identifier   string
	Capabilities map[string]interface{}
	Gateway      string
}

type gateway struct {
	Identifier   string
	Capabilities []string
	SelfDevice   string
}

type zone struct {
	Identifier int
	Name       string
	SubZones   []zone   `json:",omitempty"`
	Devices    []device `json:",omitempty"`
}
