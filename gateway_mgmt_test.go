package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func Test_loadGatewayConfigurations(t *testing.T) {
	t.Run("loads multiple gateway configurations from fixtures", func(t *testing.T) {
		wd, _ := os.Getwd()
		fixtureDirectory := strings.Join([]string{wd, "test_fixtures", "config", "gateways"}, string(os.PathSeparator))

		gwCfgs, err := loadGatewayConfigurations(fixtureDirectory)
		assert.NoError(t, err)

		assert.Len(t, gwCfgs, 2)

		assert.Equal(t, "one", gwCfgs[0].Name)
		assert.Equal(t, "two", gwCfgs[1].Name)
	})
}
