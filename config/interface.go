package config

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
)

type InterfaceConfig struct {
	Name   string `json:"-"`
	Type   string
	Config interface{}
}

func (g *InterfaceConfig) UnmarshalJSON(data []byte) error {
	if result := gjson.GetBytes(data, "Type"); !result.Exists() {
		return fmt.Errorf("failed to find interface type information")
	} else {
		g.Type = result.String()
	}

	switch g.Type {
	case "http":
		if result := gjson.GetBytes(data, "Config"); result.Exists() {
			g.Config = &HTTPInterfaceConfig{}
			return json.Unmarshal([]byte(result.Raw), g.Config)
		} else {
			return fmt.Errorf("unable to find Config stanza: %s", g.Type)
		}
	default:
		return fmt.Errorf("unknown interface configuration type: %s", g.Type)
	}
}

type HTTPInterfaceConfig struct {
	Port        int
	EnabledAPIs []string
}