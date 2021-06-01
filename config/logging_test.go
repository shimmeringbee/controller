package config

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseLogging(t *testing.T) {
	t.Run("errors if json is invalid", func(t *testing.T) {
		data := []byte(`"`)
		gw := LoggingConfig{}

		err := json.Unmarshal(data, &gw)
		assert.Error(t, err)
	})

	t.Run("errors if type is unknown", func(t *testing.T) {
		data := []byte(`{"Type":"unknown"}`)
		gw := LoggingConfig{}

		err := json.Unmarshal(data, &gw)
		assert.Error(t, err)
	})

	t.Run("stdout logger", func(t *testing.T) {
		t.Run("parses successfully", func(t *testing.T) {
			data := []byte(`{
  "Type": "stdout",
  "Config": {
    "Level": "debug",
    "Subsystems": [
      "zstack"
    ],
    "NegateSubsystems": true
  }
}`)
			gw := LoggingConfig{}

			err := json.Unmarshal(data, &gw)
			assert.NoError(t, err)

			stdoutLog, ok := gw.Config.(*StdoutLogging)
			assert.True(t, ok)

			assert.Equal(t, "debug", stdoutLog.Level)
			assert.Contains(t, stdoutLog.Subsystems, "zstack")
			assert.True(t, stdoutLog.NegateSubsystems)
		})
	})

	t.Run("file logger", func(t *testing.T) {
		t.Run("parses successfully", func(t *testing.T) {
			data := []byte(`{
  "Type": "file",
  "Config": {
    "Filename": "filename",
    "Size": 1024,
    "Count": 5,
    "Compress": true,
    "Level": "debug",
    "Subsystems": [
      "zstack"
    ],
    "NegateSubsystems": true
  }
}`)
			gw := LoggingConfig{}

			err := json.Unmarshal(data, &gw)
			assert.NoError(t, err)

			fileLog, ok := gw.Config.(*FileLogging)
			assert.True(t, ok)

			assert.Equal(t, "debug", fileLog.Level)
			assert.Contains(t, fileLog.Subsystems, "zstack")
			assert.True(t, fileLog.NegateSubsystems)

			assert.Equal(t, "filename", fileLog.Filename)
			assert.Equal(t, 1024, fileLog.Size)
			assert.Equal(t, 5, fileLog.Count)
			assert.True(t, fileLog.Compress)
		})
	})
}
