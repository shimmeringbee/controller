package main

import (
	"context"
	"github.com/gorilla/mux"
	v1 "github.com/shimmeringbee/controller/http/v1"
	"github.com/shimmeringbee/da"
	lw "github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

func main() {
	ctx := context.Background()
	l := lw.New(golog.Wrap(log.New(os.Stderr, "", log.LstdFlags)))

	l.LogInfo(ctx, "Shimmering Bee: Controller - Copyright 2019-2020 Shimmering Bee Contributors - Starting...")

	directories := enumerateDirectories(ctx, l)

	l.LogInfo(ctx, "Directory enumeration complete.", lw.Datum("directories", directories))

	gatewayCfgs, err := loadGatewayConfigurations(strings.Join([]string{directories.Config, "gateways"}, string(os.PathSeparator)))
	if err != nil {
		l.LogFatal(ctx, "Failed to load gateway configurations.", lw.Err(err))
	}

	l.LogInfo(ctx, "Loaded gateway configurations.", lw.Datum("gatewayConfigCount", len(gatewayCfgs)))

	gwMux := GatewayMux{
		deviceByIdentifier: map[string]da.Device{},
		gatewayByName:      map[string]da.Gateway{},
	}

	// Start interfaces
	r := mux.NewRouter()
	v1Router := v1.ConstructRouter(&gwMux)
	r.PathPrefix("/api/v1").Handler(http.StripPrefix("/api/v1", v1Router))

	go http.ListenAndServe(":3000", r)

	gws, err := startGateways(gatewayCfgs, &gwMux, directories)
	if err != nil {
		l.LogFatal(ctx, "Failed to start gateways.", lw.Err(err))
	}

	l.LogInfo(ctx, "Started gateways.")

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, os.Kill)

	s := <-signalCh
	l.LogInfo(ctx, "Signal received, shutting down.", lw.Datum("signal", s.String()))

	// Shutdown interfaces

	for _, gw := range gws {
		l.LogInfo(ctx, "Shutting down gateway.", lw.Datum("gateway", gw.Name))

		if err := gw.Gateway.Stop(); err != nil {
			l.LogError(ctx, "Failed to shutdown gateway.", lw.Err(err), lw.Datum("gateway", gw.Name))
		}

		gw.Shutdown()
	}

	l.LogInfo(ctx, "Shutting gateway mux.")
	gwMux.Stop()

	l.LogInfo(ctx, "Shutting complete.")
}
