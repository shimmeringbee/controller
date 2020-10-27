package config

import (
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/zigbee"
	"github.com/tidwall/gjson"
)

type Gateway struct {
	Type   string
	Config interface{}
}

func (g *Gateway) UnmarshalJSON(data []byte) error {
	if result := gjson.GetBytes(data, "Type"); !result.Exists() {
		return fmt.Errorf("failed to find gateway type information")
	} else {
		g.Type = result.String()
	}

	switch g.Type {
	case "zda":
		if result := gjson.GetBytes(data, "Config"); result.Exists() {
			g.Config = &ZDAConfig{}
			return json.Unmarshal([]byte(result.Raw), g.Config)
		} else {
			return fmt.Errorf("unable to find Config stanza: %s", g.Type)
		}
	default:
		return fmt.Errorf("unknown gateway configuration type: %s", g.Type)
	}
}

type ZDAConfig struct {
	Provider ZDAProvider
	Network  *zigbee.NetworkConfiguration
	Rules    string
}

type ZDAProvider struct {
	Type   string
	Config interface{}
}

func (g *ZDAProvider) UnmarshalJSON(data []byte) error {
	if result := gjson.GetBytes(data, "Type"); !result.Exists() {
		return fmt.Errorf("failed to find zigbee provider type information")
	} else {
		g.Type = result.String()
	}

	switch g.Type {
	case "zstack":
		if result := gjson.GetBytes(data, "Config"); result.Exists() {
			g.Config = &ZStackZDAProviderConfig{}
			return json.Unmarshal([]byte(result.Raw), g.Config)
		} else {
			return fmt.Errorf("unable to find Config stanza: %s", g.Type)
		}
	default:
		return fmt.Errorf("unknown zigbee provider configuration type: %s", g.Type)
	}
}

type ZStackZDAProviderConfig struct {
	Port struct {
		Name string
		Baud int
		RTS  bool
	}
}
