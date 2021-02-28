package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/config"
	mux2 "github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/interface/http/swagger"
	v1 "github.com/shimmeringbee/controller/interface/http/v1"
	mqtt "github.com/shimmeringbee/controller/interface/mqtt"
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

func startInterfaces(cfgs []config.InterfaceConfig, g *mux2.Mux, o *metadata.DeviceOrganiser, directories Directories, stack layers.OutputStack) ([]StartedInterface, error) {
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

func startInterface(cfg config.InterfaceConfig, g *mux2.Mux, o *metadata.DeviceOrganiser, cfgDig string, stack layers.OutputStack) (func() error, error) {
	switch gwCfg := cfg.Config.(type) {
	case *config.HTTPInterfaceConfig:
		return startHTTPInterface(*gwCfg, g, o, cfgDig, stack)
	case *config.MQTTInterfaceConfig:
		return startMQTTInterface(*gwCfg, g, o, cfgDig, stack)
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

func startHTTPInterface(cfg config.HTTPInterfaceConfig, g *mux2.Mux, o *metadata.DeviceOrganiser, cfgDir string, stack layers.OutputStack) (func() error, error) {
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

func startMQTTInterface(cfg config.MQTTInterfaceConfig, g *mux2.Mux, o *metadata.DeviceOrganiser, cfgDir string, stack layers.OutputStack) (func() error, error) {
	clientId, err := randomClientID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random client id: %w", err)
	}

	clientOptions := pahomqtt.NewClientOptions()
	clientOptions.ClientID = clientId

	i := mqtt.Interface{GatewayMux: g, GatewaySubscriber: g, DeviceOrganiser: o, OutputStack: stack}

	clientOptions.OnConnect = func(client pahomqtt.Client) {
		client.Subscribe(prefixTopic(cfg.TopicPrefix, "+"), 0, func(client pahomqtt.Client, message pahomqtt.Message) {
			i.IncomingMessage(stripPrefixTopic(cfg.TopicPrefix, message.Topic()), message.Payload())
		})

		client.Publish(prefixTopic(cfg.TopicPrefix, "controller/online"), cfg.QOS, cfg.Retained, `true`)

		i.Connected(func(prefix string, payload []byte) {
			client.Publish(prefixTopic(cfg.TopicPrefix, prefix), cfg.QOS, cfg.Retained, payload)
		}, cfg.PublishAllOnConnect)
	}

	clientOptions.SetConnectionLostHandler(func(client pahomqtt.Client, err error) {
		i.Disconnected()
	})

	clientOptions.SetWill(prefixTopic(cfg.TopicPrefix, "controller/online"), `false`, cfg.QOS, cfg.Retained)

	if cfg.Credentials != nil {
		clientOptions.SetUsername(cfg.Credentials.Username)
		clientOptions.SetPassword(cfg.Credentials.Password)
	}

	if cfg.TLS != nil {
		tlsConfig := &tls.Config{}

		if len(cfg.TLS.Cert) > 0 {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.Cert, cfg.TLS.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate/key for mqtt: %w", err)
			}

			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if len(cfg.TLS.CACert) > 0 {
			caCerts, err := ioutil.ReadFile(filepath.Clean(cfg.TLS.CACert))
			if err != nil {
				return nil, fmt.Errorf("failed to load CA TLS certificats for mqtt: %w", err)
			}

			certPool, err := x509.SystemCertPool()
			if err != nil {
				certPool = x509.NewCertPool()
			}

			certPool.AppendCertsFromPEM(caCerts)

			tlsConfig.RootCAs = certPool
		}

		clientOptions.SetTLSConfig(tlsConfig)
	}

	i.Start()

	client := pahomqtt.NewClient(clientOptions)
	client.Connect()

	return func() error {
		client.Disconnect(1500)
		i.Stop()
		return nil
	}, nil
}

func prefixTopic(topicPrefix string, topic string) string {
	if len(topicPrefix) > 0 {
		return fmt.Sprintf("%s/%s", topicPrefix, topic)
	}

	return topic
}

func stripPrefixTopic(topicPrefix string, topic string) string {
	if len(topicPrefix) > 0 {
		if strings.HasPrefix(topic, topicPrefix) {
			return topic[len(topicPrefix):]
		}
	}

	return topic
}

func randomClientID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
