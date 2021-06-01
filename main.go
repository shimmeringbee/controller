package main

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/da"
	lw "github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	ctx := context.Background()
	l := lw.New(golog.Wrap(log.New(os.Stderr, "", log.LstdFlags)))

	l.LogInfo(ctx, "Shimmering Bee: Controller - Copyright 2019-2020 Shimmering Bee Contributors - Starting...")

	directories := enumerateDirectories(ctx, l)

	l.LogInfo(ctx, "Directory enumeration complete.", lw.Datum("directories", directories))

	newLogger, err := configureLogging(filepath.Join(directories.Config, "logging"), directories.Log, l)
	if err != nil {
		l.LogFatal(ctx, "Failed to load logging configuration.", lw.Err(err))
	}

	l = newLogger

	gatewayCfgs, err := loadGatewayConfigurations(filepath.Join(directories.Config, "gateways"))
	if err != nil {
		l.LogFatal(ctx, "Failed to load gateway configurations.", lw.Err(err))
	}

	l.LogInfo(ctx, "Loaded gateway configurations.", lw.Datum("configCount", len(gatewayCfgs)))

	interfaceCfgs, err := loadInterfaceConfigurations(filepath.Join(directories.Config, "interfaces"))
	if err != nil {
		l.LogFatal(ctx, "Failed to load interface configurations.", lw.Err(err))
	}

	l.LogInfo(ctx, "Loaded interface configurations.", lw.Datum("configCount", len(interfaceCfgs)))

	l.LogInfo(ctx, "Initialising device organiser.")
	deviceOrganiser := metadata.NewDeviceOrganiser()

	shutdownDeviceOrganiser, err := initialiseDeviceOrganiser(l, directories.Data, &deviceOrganiser)
	if err != nil {
		l.LogFatal(ctx, "Failed to initialise device organiser.", lw.Err(err))
	}

	gwMux := gateway.New()

	l.LogInfo(ctx, "Linking device organiser to mux.")
	deviceOrganiserMuxCh := updateDeviceOrganiserFromMux(&deviceOrganiser)
	gwMux.Listen(deviceOrganiserMuxCh)

	outputStack := layers.PassThruStack{}

	l.LogInfo(ctx, "Starting interfaces.")
	startedInterfaces, err := startInterfaces(interfaceCfgs, gwMux, &deviceOrganiser, directories, outputStack, l)
	if err != nil {
		l.LogFatal(ctx, "Failed to start interfaces.", lw.Err(err))
	}

	l.LogInfo(ctx, "Starting gateways.")
	startedGateways, err := startGateways(gatewayCfgs, gwMux, directories, l)
	if err != nil {
		l.LogFatal(ctx, "Failed to start gateways.", lw.Err(err))
	}

	l.LogInfo(ctx, "Controller ready.")

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, os.Kill)

	s := <-signalCh
	l.LogInfo(ctx, "Signal received, shutting down.", lw.Datum("signal", s.String()))

	for _, intf := range startedInterfaces {
		l.LogInfo(ctx, "Shutting down interface.", lw.Datum("interface", intf.Name))

		if err := intf.Shutdown(); err != nil {
			l.LogError(ctx, "Failed to shutdown gateway.", lw.Err(err), lw.Datum("interface", intf.Name))
		}
	}

	for _, gw := range startedGateways {
		l.LogInfo(ctx, "Shutting down gateway.", lw.Datum("gateway", gw.Name))

		if err := gw.Gateway.Stop(); err != nil {
			l.LogError(ctx, "Failed to shutdown gateway.", lw.Err(err), lw.Datum("gateway", gw.Name))
		}

		gw.Shutdown()
	}

	l.LogInfo(ctx, "Shutting down gateway mux.")
	gwMux.Stop()

	l.LogInfo(ctx, "Shutting device organiser mux link.")
	deviceOrganiserMuxCh <- nil

	l.LogInfo(ctx, "Shutting down device organiser.")
	shutdownDeviceOrganiser()

	l.LogInfo(ctx, "Shut down complete.")
}

func initialiseDeviceOrganiser(l lw.Logger, dir string, d *metadata.DeviceOrganiser) (func(), error) {
	zoneFile := filepath.Join(dir, "zones.json")
	deviceFile := filepath.Join(dir, "devices.json")

	if err := metadata.LoadZones(zoneFile, d); err != nil {
		return func() {}, fmt.Errorf("failed to load zones: %w", err)
	}

	if err := metadata.LoadDevices(deviceFile, d); err != nil {
		return func() {}, fmt.Errorf("failed to load devices: %w", err)
	}

	if err := metadata.SaveZones(zoneFile, d); err != nil {
		return func() {}, fmt.Errorf("failed initial save of zones: %w", err)
	}

	if err := metadata.SaveDevices(deviceFile, d); err != nil {
		return func() {}, fmt.Errorf("failed initial save of devices: %w", err)
	}

	shutCh := make(chan struct{}, 1)

	go func() {
		t := time.NewTicker(1 * time.Minute)

		for {
			select {
			case <-t.C:
				if err := metadata.SaveZones(zoneFile, d); err != nil {
					l.LogError(context.Background(), "Failed to periodically save zones for device organiser.", lw.Err(err))
				}

				if err := metadata.SaveDevices(deviceFile, d); err != nil {
					l.LogError(context.Background(), "Failed to periodically save devices for device organiser.", lw.Err(err))
				}

			case <-shutCh:
				if err := metadata.SaveZones(zoneFile, d); err != nil {
					l.LogError(context.Background(), "Failed to periodically save zones for device organiser.", lw.Err(err))
				}

				if err := metadata.SaveDevices(deviceFile, d); err != nil {
					l.LogError(context.Background(), "Failed to periodically save devices for device organiser.", lw.Err(err))
				}
				return
			}
		}
	}()

	return func() {
		shutCh <- struct{}{}
	}, nil
}

func updateDeviceOrganiserFromMux(do *metadata.DeviceOrganiser) chan interface{} {
	ch := make(chan interface{}, 100)

	go func() {
		for {
			select {
			case e := <-ch:
				switch ce := e.(type) {
				case da.DeviceAdded:
					do.AddDevice(ce.Device.Identifier().String())
				case da.DeviceRemoved:
					do.RemoveDevice(ce.Device.Identifier().String())
				case nil:
					return
				}
			}
		}
	}()

	return ch
}
