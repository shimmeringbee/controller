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
		g.Config = &HTTPInterfaceConfig{}
	case "mqtt":
		g.Config = &MQTTInterfaceConfig{}
	default:
		return fmt.Errorf("unknown interface configuration type: %s", g.Type)
	}

	if result := gjson.GetBytes(data, "Config"); result.Exists() {
		return json.Unmarshal([]byte(result.Raw), g.Config)
	} else {
		return fmt.Errorf("unable to find Config stanza: %s", g.Type)
	}
}

type HTTPInterfaceConfig struct {
	Port        int
	EnabledAPIs []string
}

type MQTTInterfaceConfig struct {
	Server string

	TLS         *MQTTTLS
	Credentials *MQTTCredentials

	Retained    bool
	QOS         byte
	TopicPrefix string

	PublishStateOnConnect  bool
	PublishAggregatedState bool
	PublishIndividualState bool
}

type MQTTTLS struct {
	IgnoreSystemRootCertificates bool
	SkipCertificateVerification  bool
	Key                          string
	Cert                         string
	CACert                       string
}

type MQTTCredentials struct {
	Username string
	Password string
}
