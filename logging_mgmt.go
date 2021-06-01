package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shimmeringbee/controller/config"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/filter"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"github.com/shimmeringbee/logwrap/impl/tee"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func configureLogging(cfgDir string, logDir string, l logwrap.Logger) (logwrap.Logger, error) {
	if err := os.MkdirAll(cfgDir, DefaultDirectoryPermissions); err != nil {
		return l, fmt.Errorf("failed to ensure logging configuration directory exists: %w", err)
	}

	files, err := ioutil.ReadDir(cfgDir)
	if err != nil {
		return l, fmt.Errorf("failed to read directory listing for logging configurations: %w", err)
	}

	var logCfg []config.LoggingConfig

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		fullPath := filepath.Join(cfgDir, file.Name())
		data, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return l, fmt.Errorf("failed to read logging configuration file '%s': %w", fullPath, err)
		}

		cfg := config.LoggingConfig{
			Name: strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return l, fmt.Errorf("failed to parse logging configuration file '%s': %w", fullPath, err)
		}

		l.LogInfo(context.Background(), "Loaded logging configuration.", logwrap.Datum("name", cfg.Name), logwrap.Datum("type", cfg.Type))
		logCfg = append(logCfg, cfg)
	}

	var impls []logwrap.Impl

	for _, cfg := range logCfg {
		var logWriter io.Writer
		var baseCfg config.BaseLogging

		switch lCfg := cfg.Config.(type) {
		case *config.StdoutLogging:
			logWriter = os.Stderr
			baseCfg = lCfg.BaseLogging
		case *config.FileLogging:
			outFile := filepath.Join(logDir, lCfg.Filename)
			baseCfg = lCfg.BaseLogging

			logWriter = &lumberjack.Logger{
				Filename:   outFile,
				MaxSize:    lCfg.Size,
				MaxBackups: lCfg.Count,
				Compress:   lCfg.Compress,
			}
		}

		impl, err := constructFilter(baseCfg, golog.Wrap(log.New(logWriter, "", log.LstdFlags)))
		if err != nil {
			return l, fmt.Errorf("failed to construct filter for logging '%s': %w", cfg.Name, err)
		}

		impls = append(impls, impl)

		l.LogInfo(context.Background(), "Constructed logging.", logwrap.Datum("name", cfg.Name), logwrap.Datum("type", cfg.Type))
	}

	if len(impls) == 0 {
		l.LogWarn(context.Background(), "No logging configurations loaded, continuing with stdout/stderr only.")
		return l, nil
	}

	l.LogDebug(context.Background(), "Handing over to new logging configuration.")

	return logwrap.New(tee.Tee(impls...)), nil
}

func constructFilter(cfg config.BaseLogging, base logwrap.Impl) (logwrap.Impl, error) {
	var level logwrap.LogLevel

	if cfg.Level == "" {
		cfg.Level = "info"
	}

	switch cfg.Level {
	case "panic":
		level = logwrap.Panic
	case "fatal":
		level = logwrap.Fatal
	case "error":
		level = logwrap.Error
	case "warn":
		level = logwrap.Warn
	case "info":
		level = logwrap.Info
	case "debug":
		level = logwrap.Debug
	case "trace":
		level = logwrap.Trace
	default:
		return base, fmt.Errorf("unknown log level '%s'", cfg.Level)
	}

	return filter.Filter(base, func(message logwrap.Message) bool {
		if message.Level > level {
			return false
		}

		if len(cfg.Subsystems) == 0 {
			return true
		}

		found := false

		for _, filterSubsystem := range cfg.Subsystems {
			if filterSubsystem == message.Source {
				found = true
				break
			}
		}

		return cfg.NegateSubsystems != found
	}), nil
}
