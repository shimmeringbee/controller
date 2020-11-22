package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/config"
	v1 "github.com/shimmeringbee/controller/http/v1"
	"github.com/shimmeringbee/controller/metadata"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type StartedInterface struct {
	Name     string
	Shutdown func() error
}

func loadInterfaceConfigurations(dir string) ([]config.InterfaceConfig, error) {
	if err := os.MkdirAll(dir, DefaultDirectoryPermissions); err != nil {
		return nil, fmt.Errorf("failed to ensure interface configuration directory exists: %w", err)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory listing for interface configurations: %w", err)
	}

	var retCfgs []config.InterfaceConfig

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		fullPath := fmt.Sprintf("%s%s%s", dir, string(os.PathSeparator), file.Name())
		data, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read interface configuration file '%s': %w", fullPath, err)
		}

		cfg := config.InterfaceConfig{
			Name: strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse interface configuration file '%s': %w", fullPath, err)
		}

		retCfgs = append(retCfgs, cfg)
	}

	return retCfgs, nil
}

func startInterfaces(cfgs []config.InterfaceConfig, g *GatewayMux, o *metadata.DeviceOrganiser, directories Directories) ([]StartedInterface, error) {
	var retGws []StartedInterface

	for _, cfg := range cfgs {
		dataDir := strings.Join([]string{directories.Data, "interfaces", cfg.Name}, string(os.PathSeparator))

		if err := os.MkdirAll(dataDir, DefaultDirectoryPermissions); err != nil {
			return nil, fmt.Errorf("failed to create interface data directory '%s': %w", dataDir, err)
		}

		if shutdown, err := startInterface(cfg, g, o, dataDir); err != nil {
			return nil, fmt.Errorf("failed to start interface '%s': %w", cfg.Name, err)
		} else {
			retGws = append(retGws, StartedInterface{
				Name:     cfg.Name,
				Shutdown: shutdown,
			})
		}
	}

	return retGws, nil
}

func startInterface(cfg config.InterfaceConfig, g *GatewayMux, o *metadata.DeviceOrganiser, cfgDig string) (func() error, error) {
	switch gwCfg := cfg.Config.(type) {
	case *config.HTTPInterfaceConfig:
		return startHTTPInterface(*gwCfg, g, o, cfgDig)
	default:
		return nil, fmt.Errorf("unknown gateway type loaded: %s", cfg.Type)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}

	return false
}

func startHTTPInterface(cfg config.HTTPInterfaceConfig, g *GatewayMux, o *metadata.DeviceOrganiser, cfgDig string) (func() error, error) {
	r := mux.NewRouter()

	if containsString(cfg.EnabledAPIs, "v1") {
		v1Router := v1.ConstructRouter(g, o)
		r.PathPrefix("/api/v1").Handler(http.StripPrefix("/api/v1", v1Router))
	}

	bindAddress := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: bindAddress, Handler: r}

	go srv.ListenAndServe()

	return func() error {
		return srv.Shutdown(context.Background())
	}, nil
}