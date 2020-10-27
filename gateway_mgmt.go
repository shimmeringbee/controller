package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/config"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zda/capability/has_product_information"
	"github.com/shimmeringbee/zda/capability/on_off"
	"github.com/shimmeringbee/zda/capability/pressure_sensor"
	"github.com/shimmeringbee/zda/capability/relative_humidity_sensor"
	"github.com/shimmeringbee/zda/capability/temperature_sensor"
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

func startGateways(cfgs []config.Gateway, mux *GatewayMux, directories Directories) ([]StartedGateway, error) {
	var retGws []StartedGateway

	for _, cfg := range cfgs {
		cfgDig := strings.Join([]string{directories.Data, "gateways", cfg.Name}, string(os.PathSeparator))

		if err := os.MkdirAll(cfgDig, DefaultDirectoryPermissions); err != nil {
			return nil, fmt.Errorf("failed to create gateway config directory '%s': %w", cfgDig, err)
		}

		if gw, shutdown, err := startGateway(cfg, cfgDig); err != nil {
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

func startGateway(cfg config.Gateway, cfgDig string) (da.Gateway, func(), error) {
	switch gwCfg := cfg.Config.(type) {
	case *config.ZDAConfig:
		return startZDAGateway(*gwCfg, cfgDig)
	default:
		return nil, nil, fmt.Errorf("unknown gateway type loaded: %s", cfg.Type)
	}
}

func startZDAGateway(cfg config.ZDAConfig, cfgDig string) (da.Gateway, func(), error) {
	provider, providerShut, err := startZigbeeProvider(cfg.Provider, *cfg.Network, cfgDig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start zigbee provider: %w", err)
	}

	var r *rules.Rule
	if len(cfg.Rules) > 0 {
		r, err = loadZDARules(cfg.Rules)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load zda rules '%s': %w", cfg.Rules, err)
		}
	}

	gw := zda.New(provider, r)
	gw.CapabilityManager.Add(&has_product_information.Implementation{})
	gw.CapabilityManager.Add(&on_off.Implementation{})
	gw.CapabilityManager.Add(&temperature_sensor.Implementation{})
	gw.CapabilityManager.Add(&relative_humidity_sensor.Implementation{})
	gw.CapabilityManager.Add(&pressure_sensor.Implementation{})

	if err := gw.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start zda: %w", err)
	}

	zdaStateFile := strings.Join([]string{cfgDig, "zda_state.json"}, string(os.PathSeparator))

	if err := loadZDAState(gw, zdaStateFile); err != nil {
		return nil, nil, fmt.Errorf("failed to load zda state: %w", err)
	}

	if err := saveZDAState(gw, zdaStateFile); err != nil {
		return nil, nil, fmt.Errorf("failed to save/create zda state: %w", err)
	}

	shutCh := make(chan struct{}, 1)

	go func() {
		t := time.NewTicker(1 * time.Minute)

		for {
			select {
			case <-t.C:
				if err := loadZDAState(gw, zdaStateFile); err != nil {
				}
			case <-shutCh:
				if err := saveZDAState(gw, zdaStateFile); err != nil {
				}
				return
			}
		}
	}()

	return gw, func() {
		providerShut()
		shutCh <- struct{}{}
	}, nil
}

func saveZDAState(gw *zda.ZigbeeGateway, file string) error {
	state := gw.SaveState()

	if data, err := json.MarshalIndent(state, "", "\t"); err != nil {
		return fmt.Errorf("failed to marshal zda state: %w", err)
	} else {
		if err := ioutil.WriteFile(file, data, DefaultFilePermissions); err != nil {
			return fmt.Errorf("failed to save zda state: %w", err)
		}
	}

	return nil
}

func loadZDAState(gw *zda.ZigbeeGateway, file string) error {
	if _, err := os.Stat(file); err == nil {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to load read zda state: %w", err)
		}

		state, err := zda.JSONUnmarshalState(gw, data)
		if err != nil {
			return fmt.Errorf("failed to parse zda state: %w", err)
		}

		if err := gw.LoadState(state); err != nil {
			return fmt.Errorf("failed to load zda state: %w", err)
		}
	}

	return nil
}

func loadZDARules(file string) (*rules.Rule, error) {
	if _, err := os.Stat(file); err != nil {
		if err == os.ErrNotExist {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to find ZDA rules: %w", err)
	}

	var rule rules.Rule

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read ZDA rules: %w", err)
	}

	if err := json.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("failed to parse ZDA rules: %w", err)
	}

	return &rule, nil
}

func startZigbeeProvider(providerCfg config.ZDAProvider, network zigbee.NetworkConfiguration, cfgDig string) (zigbee.Provider, func(), error) {
	switch pvdCfg := providerCfg.Config.(type) {
	case *config.ZStackProvider:
		return startZStackProvider(*pvdCfg, network, cfgDig)
	default:
		return nil, nil, fmt.Errorf("unknown provider type loaded: %s", providerCfg.Type)
	}
}

func startZStackProvider(cfg config.ZStackProvider, network zigbee.NetworkConfiguration, cfgDig string) (zigbee.Provider, func(), error) {
	port, err := serial.Open(cfg.Port.Name, &serial.Mode{BaudRate: cfg.Port.Baud})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open serial port for zstack '%s': %w", cfg.Port.Name, err)
	}

	if err := port.SetRTS(cfg.Port.RTS); err != nil {
		return nil, nil, fmt.Errorf("failed to set RTS on serial port for zstack '%s': %w", cfg.Port.Name, err)
	}

	nodeCacheFile := strings.Join([]string{cfgDig, "zstack_node_cache.json"}, string(os.PathSeparator))
	nodeCache := zstack.NewNodeTable()

	if err := loadZStackNodeCache(nodeCache, nodeCacheFile); err != nil {
		return nil, nil, fmt.Errorf("failed to load node cache for zstack: %w", err)
	}

	if err := saveZStackNodeCache(nodeCache, nodeCacheFile); err != nil {
		return nil, nil, fmt.Errorf("failed to save/create node cache for zstack: %w", err)
	}

	z := zstack.New(port, nodeCache)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := z.Initialise(ctx, network); err != nil {
		port.Close()

		return nil, nil, fmt.Errorf("failed to initialise zstack: %w", err)
	}

	shutCh := make(chan struct{}, 1)

	go func() {
		t := time.NewTicker(1 * time.Minute)

		for {
			select {
			case <-t.C:
				if err := saveZStackNodeCache(nodeCache, nodeCacheFile); err != nil {
				}
			case <-shutCh:
				if err := saveZStackNodeCache(nodeCache, nodeCacheFile); err != nil {
				}
				return
			}
		}
	}()

	return z, func() {
		shutCh <- struct{}{}
	}, nil
}

func saveZStackNodeCache(cache *zstack.NodeTable, file string) error {
	if data, err := json.MarshalIndent(cache.Nodes(), "", "\t"); err != nil {
		return fmt.Errorf("failed to marshal zstack node cache: %w", err)
	} else {
		if err := ioutil.WriteFile(file, data, DefaultFilePermissions); err != nil {
			return fmt.Errorf("failed to save zstack node cache: %w", err)
		}
	}

	return nil
}

func loadZStackNodeCache(cache *zstack.NodeTable, file string) error {
	var nodes []zigbee.Node

	if _, err := os.Stat(file); err == nil {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to load zstack node cache: %w", err)
		}

		err = json.Unmarshal(data, &nodes)
		if err != nil {
			return fmt.Errorf("failed to parse zstack node cache: %w", err)
		}

		cache.Load(nodes)
	}

	return nil
}

func loadGatewayConfigurations(dir string) ([]config.Gateway, error) {
	if err := os.MkdirAll(dir, DefaultDirectoryPermissions); err != nil {
		return nil, fmt.Errorf("failed to ensure gateway configuration directory exists: %w", err)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory listing for gateway configurations: %w", err)
	}

	var retCfgs []config.Gateway

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		fullPath := fmt.Sprintf("%s%s%s", dir, string(os.PathSeparator), file.Name())
		data, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read gateway configuration file '%s': %w", fullPath, err)
		}

		cfg := config.Gateway{
			Name: strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse gateway configuration file '%s': %w", fullPath, err)
		}

		retCfgs = append(retCfgs, cfg)
	}

	return retCfgs, nil
}