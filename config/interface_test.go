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
			data := []byte(`{"Type":"http","Config":{"Port":3000,"EnabledAPIs":["v1"]}}`)
			gw := InterfaceConfig{}

			err := json.Unmarshal(data, &gw)
			assert.NoError(t, err)

			httpInt, ok := gw.Config.(*HTTPInterfaceConfig)
			assert.True(t, ok)

			assert.Equal(t, 3000, httpInt.Port)
			assert.Contains(t, httpInt.EnabledAPIs, "v1")
		})
	})
}
