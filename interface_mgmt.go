package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/config"
	"github.com/shimmeringbee/controller/http/swagger"
	v1 "github.com/shimmeringbee/controller/http/v1"
	"github.com/shimmeringbee/controller/layers"
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

		fullPath := filepath.Join(dir, file.Name())
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

func startInterfaces(cfgs []config.InterfaceConfig, g *GatewayMux, o *metadata.DeviceOrganiser, directories Directories, stack layers.OutputStack) ([]StartedInterface, error) {
	var retGws []StartedInterface

	for _, cfg := range cfgs {
		dataDir := filepath.Join(directories.Data, "interfaces", cfg.Name)

		if err := os.MkdirAll(dataDir, DefaultDirectoryPermissions); err != nil {
			return nil, fmt.Errorf("failed to create interface data directory '%s': %w", dataDir, err)
		}

		if shutdown, err := startInterface(cfg, g, o, dataDir, stack); err != nil {
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

func startInterface(cfg config.InterfaceConfig, g *GatewayMux, o *metadata.DeviceOrganiser, cfgDig string, stack layers.OutputStack) (func() error, error) {
	switch gwCfg := cfg.Config.(type) {
	case *config.HTTPInterfaceConfig:
		return startHTTPInterface(*gwCfg, g, o, cfgDig, stack)
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

func startHTTPInterface(cfg config.HTTPInterfaceConfig, g *GatewayMux, o *metadata.DeviceOrganiser, cfgDig string, stack layers.OutputStack) (func() error, error) {
	r := mux.NewRouter()

	if containsString(cfg.EnabledAPIs, "swagger") {
		swaggerRouter := swagger.ConstructRouter()
		// This route is needed because the redirect provided by http.FileServer is incorrect due to the http.StripPrefix
		// below. As such we need to perform a manual redirect before the http.FileServer has the opportunity. Also use
		// a temporary redirect rather than the permanent used by http.FileServer.
		r.Path("/swagger").Handler(http.RedirectHandler("/swagger/", http.StatusTemporaryRedirect))
		r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger", swaggerRouter))
	}

	if containsString(cfg.EnabledAPIs, "v1") {
		v1Router := v1.ConstructRouter(g, o, stack)
		// Use http.StripPrefix to obscure the real path from the v1 api code, though this will cause issues if we
		// ever issue redirects from the API.
		r.PathPrefix("/api/v1").Handler(http.StripPrefix("/api/v1", v1Router))
	}

	bindAddress := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: bindAddress, Handler: r}

	go srv.ListenAndServe()

	return func() error {
		return srv.Shutdown(context.Background())
	}, nil
}
