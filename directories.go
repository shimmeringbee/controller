package main

import (
	"context"
	"flag"
	"github.com/peterbourgon/ff/v3"
	"github.com/shimmeringbee/logwrap"
	"os"
	"path/filepath"
)

const DefaultDirectoryPermissions = 0700

type Directories struct {
	Config string
	Data   string
	Log    string
}

func enumerateDirectories(ctx context.Context, l logwrap.Logger) Directories {
	fs := flag.NewFlagSet("controller", flag.ExitOnError)

	defaultConfigDirectory, err := defaultDirectory("config")
	if err != nil {
		l.LogFatal(ctx, "Failed to construct default configuration directory.", logwrap.Err(err))
	}

	defaultDataDirectory, err := defaultDirectory("data")
	if err != nil {
		l.LogFatal(ctx, "Failed to construct default data directory.", logwrap.Err(err))
	}

	defaultLogDirectory, err := defaultDirectory("log")
	if err != nil {
		l.LogFatal(ctx, "Failed to construct default log directory.", logwrap.Err(err))
	}

	configDirectory := fs.String("config-directory", defaultConfigDirectory, "location of configuration files")
	dataDirectory := fs.String("data-directory", defaultDataDirectory, "location of data files")
	logDirectory := fs.String("log-directory", defaultLogDirectory, "location of log files")

	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		l.LogFatal(ctx, "Failed to parse environment/command line arguments.", logwrap.Err(err))
	}

	if err := os.MkdirAll(*configDirectory, DefaultDirectoryPermissions); err != nil {
		l.LogFatal(ctx, "Failed to initialise configuration directory.", logwrap.Err(err))
	}

	if err := os.MkdirAll(*dataDirectory, DefaultDirectoryPermissions); err != nil {
		l.LogFatal(ctx, "Failed to initialise data directory.", logwrap.Err(err))
	}

	if err := os.MkdirAll(*logDirectory, DefaultDirectoryPermissions); err != nil {
		l.LogFatal(ctx, "Failed to initialise log directory.", logwrap.Err(err))
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
		return filepath.Join(configDir, "shimmeringbee", "controller", t), nil
	}
}
