package config

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
)

type LoggingConfig struct {
	Name   string `json:"-"`
	Type   string
	Config any
}

func (g *LoggingConfig) UnmarshalJSON(data []byte) error {
	if result := gjson.GetBytes(data, "Type"); !result.Exists() {
		return fmt.Errorf("failed to find logging type information")
	} else {
		g.Type = result.String()
	}

	switch g.Type {
	case "stdout":
		g.Config = &StdoutLogging{}
	case "file":
		g.Config = &FileLogging{}
	default:
		return fmt.Errorf("unknown logging configuration type: %s", g.Type)
	}

	if result := gjson.GetBytes(data, "Config"); result.Exists() {
		return json.Unmarshal([]byte(result.Raw), g.Config)
	} else {
		return fmt.Errorf("unable to find Config stanza: %s", g.Type)
	}
}

type BaseLogging struct {
	Level string

	NegateSubsystems bool
	Subsystems       []string
}

type StdoutLogging struct {
	BaseLogging
}

type FileLogging struct {
	BaseLogging

	Filename string
	Size     int
	Count    int
	Compress bool
}
