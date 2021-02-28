package config

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseInterface(t *testing.T) {
	t.Run("errors if json is invalid", func(t *testing.T) {
		data := []byte(`"`)
		gw := InterfaceConfig{}

		err := json.Unmarshal(data, &gw)
		assert.Error(t, err)
	})

	t.Run("errors if type is unknown", func(t *testing.T) {
		data := []byte(`{"Type":"unknown"}`)
		gw := InterfaceConfig{}

		err := json.Unmarshal(data, &gw)
		assert.Error(t, err)
	})

	t.Run("http gateway", func(t *testing.T) {
		t.Run("parses successfully", func(t *testing.T) {
			data := []byte(`{
  "Type": "http",
  "Config": {
    "Port": 3000,
    "EnabledAPIs": [
      "v1"
    ]
  }
}`)
			gw := InterfaceConfig{}

			err := json.Unmarshal(data, &gw)
			assert.NoError(t, err)

			httpInt, ok := gw.Config.(*HTTPInterfaceConfig)
			assert.True(t, ok)

			assert.Equal(t, 3000, httpInt.Port)
			assert.Contains(t, httpInt.EnabledAPIs, "v1")
		})
	})

	t.Run("mqtt gateway", func(t *testing.T) {
		t.Run("parses successfully", func(t *testing.T) {
			data := []byte(`{
  "Type": "mqtt",
  "Config": {
    "Server": "tcp://127.0.0.1:1883",
    "TLS": {
      "Key": "key.pem",
      "Cert": "cert.pem",
      "CACert": "cacert.pem"
    },
    "Credentials": {
      "Username": "user",
      "Password": "pass"
    },
    "Retained": true,
    "QOS": 2
  }
}`)
			gw := InterfaceConfig{}

			err := json.Unmarshal(data, &gw)
			assert.NoError(t, err)

			mqttInt, ok := gw.Config.(*MQTTInterfaceConfig)
			assert.True(t, ok)

			assert.Equal(t, "tcp://127.0.0.1:1883", mqttInt.Server)

			assert.Equal(t, "key.pem", mqttInt.TLS.Key)
			assert.Equal(t, "cert.pem", mqttInt.TLS.Cert)
			assert.Equal(t, "cacert.pem", mqttInt.TLS.CACert)

			assert.Equal(t, "user", mqttInt.Credentials.Username)
			assert.Equal(t, "pass", mqttInt.Credentials.Password)

			assert.True(t, mqttInt.Retained)
			assert.Equal(t, uint8(2), mqttInt.QOS)
		})
	})
}
