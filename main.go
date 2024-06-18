package main

import (
	"context"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/state"
	"github.com/shimmeringbee/da"
	lw "github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"github.com/shimmeringbee/persistence"
	"github.com/shimmeringbee/persistence/impl/file"
	"log"
	"os"
	"os/signal"
	"path/filepath"
)

func main() {
	ctx := context.Background()
	l := lw.New(golog.Wrap(log.New(os.Stderr, "", log.LstdFlags)))

	l.LogInfo(ctx, "Shimmering Bee: Controller - Copyright 2019-2020 Shimmering Bee Contributors - Starting...")

	directories := enumerateDirectories(ctx, l)

	l.LogInfo(ctx, "Directory enumeration complete.", lw.Datum("directories", directories))

	l.LogInfo(ctx, "Persisted data initialising.")
	section := file.New(directories.Data)

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
	deviceOrganiser := state.NewDeviceOrganiser(section.Section("Organiser"))

	eventbus := state.NewEventBus()
	gwMux := state.NewGatewayMux(eventbus)

	l.LogInfo(ctx, "Linking device organiser to mux.")
	deviceOrganiserMuxCh := updateDeviceOrganiserFromMux(&deviceOrganiser)
	eventbus.Subscribe(deviceOrganiserMuxCh)

	outputStack := layers.PassThruStack{}

	l.LogInfo(ctx, "Starting interfaces.")
	startedInterfaces, err := startInterfaces(interfaceCfgs, gwMux, eventbus, &deviceOrganiser, outputStack, l)
	if err != nil {
		l.LogFatal(ctx, "Failed to start interfaces.", lw.Err(err))
	}

	l.LogInfo(ctx, "Starting gateways.")
	startedGateways, err := startGateways(gatewayCfgs, gwMux, l, section.Section("Gateway"))
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

		if err := gw.Gateway.Stop(ctx); err != nil {
			l.LogError(ctx, "Failed to shutdown gateway.", lw.Err(err), lw.Datum("gateway", gw.Name))
		}

		gw.Shutdown()
	}

	l.LogInfo(ctx, "Shutting down gateway mux.")
	gwMux.Stop()

	l.LogInfo(ctx, "Shutting device organiser mux link.")
	deviceOrganiserMuxCh <- nil

	if syncer, ok := section.(persistence.Syncer); ok {
		l.LogInfo(ctx, "Syncing persistence.")
		syncer.Sync()
	}

	l.LogInfo(ctx, "Shut down complete.")
}

func updateDeviceOrganiserFromMux(do *state.DeviceOrganiser) chan interface{} {
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
