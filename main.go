package main

import (
	"context"
	"flag"
	"github.com/peterbourgon/ff/v3"
	lw "github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"log"
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

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, os.Kill)

	// Load configuration

	// Start interfaces

	// Start gateways

	s := <-signalCh
	l.LogInfo(ctx, "Shutting down.", lw.Datum("signal", s.String()))

	// Shutdown interfaces

	// Shutdown gateways
}

type Directories struct {
	Config string
	Data   string
	Log    string
}

func enumerateDirectories(ctx context.Context, l lw.Logger) Directories {
	fs := flag.NewFlagSet("controller", flag.ExitOnError)

	defaultConfigDirectory, err := defaultDirectory("config")
	if err != nil {
		l.LogFatal(ctx, "Failed to construct default configuration directory.", lw.Err(err))
	}

	defaultDataDirectory, err := defaultDirectory("data")
	if err != nil {
		l.LogFatal(ctx, "Failed to construct default data directory.", lw.Err(err))
	}

	defaultLogDirectory, err := defaultDirectory("log")
	if err != nil {
		l.LogFatal(ctx, "Failed to construct default log directory.", lw.Err(err))
	}

	configDirectory := fs.String("config-directory", defaultConfigDirectory, "location of configuration files")
	dataDirectory := fs.String("data-directory", defaultDataDirectory, "location of data files")
	logDirectory := fs.String("log-directory", defaultLogDirectory, "location of log files")

	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		l.LogFatal(ctx, "Failed to parse environment/command line arguments.", lw.Err(err))
	}

	if err := os.MkdirAll(*configDirectory, 0700); err != nil {
		l.LogFatal(ctx, "Failed to initialise configuration directory.", lw.Err(err))
	}

	if err := os.MkdirAll(*dataDirectory, 0700); err != nil {
		l.LogFatal(ctx, "Failed to initialise data directory.", lw.Err(err))
	}

	if err := os.MkdirAll(*logDirectory, 0700); err != nil {
		l.LogFatal(ctx, "Failed to initialise log directory.", lw.Err(err))
	}

	return Directories{
		Config: *configDirectory,
		Data:   *dataDirectory,
		Log:    *logDirectory,
	}
}

func defaultDirectory(t string) (string, error) {
	if configDir, err := os.UserConfigDir(); err != nil {
		return "", err
	} else {
		return strings.Join([]string{configDir, "shimmeringbee", "controller", t}, string(os.PathSeparator)), nil
	}
}
