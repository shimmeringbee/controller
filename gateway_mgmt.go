package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/config"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/nest"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"
	"go.bug.st/serial.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type StartedGateway struct {
	Name     string
	Gateway  da.Gateway
	Shutdown func()
}

func startGateways(cfgs []config.GatewayConfig, mux *state.GatewayMux, directories Directories, l logwrap.Logger, s persistence.Section) ([]StartedGateway, error) {
	var retGws []StartedGateway

	for _, cfg := range cfgs {
		dataDir := filepath.Join(directories.Data, "gateways", cfg.Name)

		if err := os.MkdirAll(dataDir, DefaultDirectoryPermissions); err != nil {
			return nil, fmt.Errorf("failed to create gateway data directory '%s': %w", dataDir, err)
		}

		gwSection := s.Section(cfg.Name)

		if gw, shutdown, err := startGateway(cfg, dataDir, l, gwSection); err != nil {
			return nil, fmt.Errorf("failed to start gateway '%s': %w", cfg.Name, err)
		} else {
			mux.Add(cfg.Name, gw)
			retGws = append(retGws, StartedGateway{
				Gateway:  gw,
				Name:     cfg.Name,
				Shutdown: shutdown,
			})
		}
	}

	return retGws, nil
}

func startGateway(cfg config.GatewayConfig, cfgDig string, l logwrap.Logger, s persistence.Section) (da.Gateway, func(), error) {
	wl := logwrap.New(nest.Wrap(l))
	wl.AddOptionsToLogger(logwrap.Datum("gateway", cfg.Name))

	switch gwCfg := cfg.Config.(type) {
	case *config.ZDAConfig:
		wl.AddOptionsToLogger(logwrap.Source("zda"))
		return startZDAGateway(*gwCfg, cfgDig, wl, s)
	default:
		return nil, nil, fmt.Errorf("unknown gateway type loaded: %s", cfg.Type)
	}
}

func startZDAGateway(cfg config.ZDAConfig, cfgDig string, l logwrap.Logger, s persistence.Section) (da.Gateway, func(), error) {
	provider, providerShut, err := startZigbeeProvider(cfg.Provider, *cfg.Network, cfgDig, l, s)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start zigbee provider: %w", err)
	}

	re := rules.New()
	if err = re.LoadFS(rules.Embedded); err != nil {
		panic(err)
	}

	if len(cfg.Rules) > 0 {
		rfs := os.DirFS(cfg.Rules)

		if err = re.LoadFS(rfs); err != nil {
			panic(err)
		}
	}

	if err = re.CompileRules(); err != nil {
		panic(err)
	}

	ctx := context.Background()

	wl := logwrap.New(nest.Wrap(l))
	wl.AddOptionsToLogger(logwrap.Source("zda"))

	gw := zda.New(ctx, s.Section("ZDA"), provider, re)
	gw.WithLogWrapLogger(wl)

	if err := gw.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start zda: %w", err)
	}

	return gw, func() {
		gw.Stop(ctx)
		providerShut()
	}, nil
}

func startZigbeeProvider(providerCfg config.ZDAProvider, network zigbee.NetworkConfiguration, cfgDig string, l logwrap.Logger, s persistence.Section) (zigbee.Provider, func(), error) {
	switch pvdCfg := providerCfg.Config.(type) {
	case *config.ZStackProvider:
		return startZStackProvider(*pvdCfg, network, cfgDig, l, s)
	default:
		return nil, nil, fmt.Errorf("unknown provider type loaded: %s", providerCfg.Type)
	}
}

func startZStackProvider(cfg config.ZStackProvider, network zigbee.NetworkConfiguration, _ string, l logwrap.Logger, s persistence.Section) (zigbee.Provider, func(), error) {
	port, err := serial.Open(cfg.Port.Name, &serial.Mode{BaudRate: cfg.Port.Baud})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open serial port for zstack '%s': %w", cfg.Port.Name, err)
	}

	wl := logwrap.New(nest.Wrap(l))
	wl.AddOptionsToLogger(logwrap.Source("zstack"))

	z := zstack.New(port, s.Section("ZStack"))
	z.WithLogWrapLogger(wl)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := z.Initialise(ctx, network); err != nil {
		port.Close()

		return nil, nil, fmt.Errorf("failed to initialise zstack: %w", err)
	}

	return z, func() {
		z.Stop()
	}, nil
}

func loadGatewayConfigurations(dir string) ([]config.GatewayConfig, error) {
	if err := os.MkdirAll(dir, DefaultDirectoryPermissions); err != nil {
		return nil, fmt.Errorf("failed to ensure gateway configuration directory exists: %w", err)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory listing for gateway configurations: %w", err)
	}

	var retCfgs []config.GatewayConfig

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		fullPath := filepath.Join(dir, file.Name())
		data, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read gateway configuration file '%s': %w", fullPath, err)
		}

		cfg := config.GatewayConfig{
			Name: strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse gateway configuration file '%s': %w", fullPath, err)
		}

		retCfgs = append(retCfgs, cfg)
	}

	return retCfgs, nil
}
