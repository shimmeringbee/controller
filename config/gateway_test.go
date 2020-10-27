package config

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseGateway(t *testing.T) {
	t.Run("errors if json is invalid", func(t *testing.T) {
		data := []byte(`"`)
		gw := Gateway{}

		err := json.Unmarshal(data, &gw)
		assert.Error(t, err)
	})

	t.Run("errors if type is unknown", func(t *testing.T) {
		data := []byte(`{"Type":"unknown"}`)
		gw := Gateway{}

		err := json.Unmarshal(data, &gw)
		assert.Error(t, err)
	})

	t.Run("zda gateway", func(t *testing.T) {
		t.Run("errors if provider type is not recognised", func(t *testing.T) {
			data := []byte(`{"Type":"zda","Config":{"Provider":{"Type":"unknown"}}}`)
			gw := Gateway{}

			err := json.Unmarshal(data, &gw)
			assert.Error(t, err)
		})

		t.Run("parses successfully if zstack is found", func(t *testing.T) {
			data := []byte(`{"Type":"zda","Config":{"Provider":{"Type":"zstack","Config":{}},"Network":{"PANID":1}}}`)
			gw := Gateway{}

			err := json.Unmarshal(data, &gw)
			assert.NoError(t, err)

			zdaGw, ok := gw.Config.(*ZDAConfig)
			assert.True(t, ok)

			_, ok = zdaGw.Provider.Config.(*ZStackZDAProviderConfig)
			assert.True(t, ok)
		})
	})
}
