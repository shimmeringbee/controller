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
	gorillamux "github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/config"
	"github.com/shimmeringbee/controller/gateway"
	"github.com/shimmeringbee/controller/interface/http/swagger"
	"github.com/shimmeringbee/controller/interface/http/v1"
	"github.com/shimmeringbee/controller/interface/mqtt"
	"github.com/shimmeringbee/controller/layers"
	"github.com/shimmeringbee/controller/metadata"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/nest"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type StartedInterface struct {
	Name     string
	Shutdown func() error
}

const DefaultMQTTEventDuration = 1 * time.Second

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

func startInterfaces(cfgs []config.InterfaceConfig, g *gateway.Mux, o *metadata.DeviceOrganiser, directories Directories, stack layers.OutputStack, l logwrap.Logger) ([]StartedInterface, error) {
	var retGws []StartedInterface

	for _, cfg := range cfgs {
		dataDir := filepath.Join(directories.Data, "interfaces", cfg.Name)

		if err := os.MkdirAll(dataDir, DefaultDirectoryPermissions); err != nil {
			return nil, fmt.Errorf("failed to create interface data directory '%s': %w", dataDir, err)
		}

		if shutdown, err := startInterface(cfg, g, o, dataDir, stack, l); err != nil {
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

func startInterface(cfg config.InterfaceConfig, g *gateway.Mux, o *metadata.DeviceOrganiser, cfgDig string, stack layers.OutputStack, l logwrap.Logger) (func() error, error) {
	wl := logwrap.New(nest.Wrap(l))
	wl.AddOptionsToLogger(logwrap.Datum("interface", cfg.Name))

	switch gwCfg := cfg.Config.(type) {
	case *config.HTTPInterfaceConfig:
		wl.AddOptionsToLogger(logwrap.Source("http"))
		return startHTTPInterface(*gwCfg, g, o, cfgDig, stack, wl)
	case *config.MQTTInterfaceConfig:
		wl.AddOptionsToLogger(logwrap.Source("mqtt"))
		return startMQTTInterface(*gwCfg, g, o, cfgDig, stack, wl)
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

func startHTTPInterface(cfg config.HTTPInterfaceConfig, g *gateway.Mux, o *metadata.DeviceOrganiser, cfgDir string, stack layers.OutputStack, l logwrap.Logger) (func() error, error) {
	r := gorillamux.NewRouter()

	if containsString(cfg.EnabledAPIs, "swagger") {
		l.LogInfo(context.Background(), "Mounting swagger endpoint on /swagger.")

		swaggerRouter := swagger.ConstructRouter()
		// This route is needed because the redirect provided by http.FileServer is incorrect due to the http.StripPrefix
		// below. As such we need to perform a manual redirect before the http.FileServer has the opportunity. Also use
		// a temporary redirect rather than the permanent used by http.FileServer.
		r.Path("/swagger").Handler(http.RedirectHandler("/swagger/", http.StatusTemporaryRedirect))
		r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger", swaggerRouter))
	}

	if containsString(cfg.EnabledAPIs, "v1") {
		l.LogInfo(context.Background(), "Mounting v1 API endpoint on /api/v1.")

		v1Router := v1.ConstructRouter(g, o, stack, l)
		// Use http.StripPrefix to obscure the real path from the v1 api code, though this will cause issues if we
		// ever issue redirects from the API.
		r.PathPrefix("/api/v1").Handler(http.StripPrefix("/api/v1", v1Router))
	}

	bindAddress := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: bindAddress, Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			l.LogError(context.Background(), "Failed to start http server.", logwrap.Err(err))
		}
	}()

	return func() error {
		return srv.Shutdown(context.Background())
	}, nil
}

func awaitToken(ctx context.Context, token pahomqtt.Token) error {
	select {
	case <-token.Done():
		return token.Error()
	case <-ctx.Done():
		return context.DeadlineExceeded
	}
}

func startMQTTInterface(cfg config.MQTTInterfaceConfig, g *gateway.Mux, o *metadata.DeviceOrganiser, cfgDir string, stack layers.OutputStack, l logwrap.Logger) (func() error, error) {
	clientId, err := randomClientID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random client id: %w", err)
	}

	l.LogInfo(context.Background(), "Constructing new MQTT client.", logwrap.Datum("clientId", clientId), logwrap.Datum("server", cfg.Server))

	clientOptions := pahomqtt.NewClientOptions()
	clientOptions.ClientID = clientId

	if url, err := url2.Parse(cfg.Server); err != nil {
		l.LogError(context.Background(), "Failed to parse MQTT server URL.", logwrap.Err(err))
		return nil, err
	} else {
		clientOptions.Servers = []*url2.URL{url}
	}

	i := mqtt.Interface{GatewayMux: g, GatewaySubscriber: g, DeviceOrganiser: o, OutputStack: stack, Logger: l, Publisher: mqtt.EmptyPublisher, PublishStateOnConnect: cfg.PublishStateOnConnect, PublishIndividualState: cfg.PublishIndividualState, PublishAggregatedState: cfg.PublishAggregatedState}

	lastWillTopic := prefixTopic(cfg.TopicPrefix, "controller/online")

	clientOptions.OnConnect = func(client pahomqtt.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultMQTTEventDuration)
		defer cancel()

		l.LogInfo(context.Background(), "MQTT client successfully connected.", logwrap.Datum("clientId", clientId), logwrap.Datum("server", cfg.Server))

		subTopic := prefixTopic(cfg.TopicPrefix, "+")
		subscribeToken := client.Subscribe(subTopic, 0, func(client pahomqtt.Client, message pahomqtt.Message) {
			ctx, cancel := context.WithTimeout(context.Background(), DefaultMQTTEventDuration)
			defer cancel()

			if i.IncomingMessage(ctx, stripPrefixTopic(cfg.TopicPrefix, message.Topic()), message.Payload()) != nil {
				l.LogError(ctx, "Failed to handle incoming message.", logwrap.Datum("topic", message.Topic()), logwrap.Err(err))
			}
		})

		if err := awaitToken(ctx, subscribeToken); err != nil {
			l.LogError(ctx, "Failed to subscribe to topic in MQTT.", logwrap.Datum("topic", subTopic), logwrap.Err(err))
		}

		client.Publish(lastWillTopic, cfg.QOS, cfg.Retained, `true`)

		if err := i.Connected(context.Background(), func(ctx context.Context, topic string, payload []byte) error {
			prefixedTopic := prefixTopic(cfg.TopicPrefix, topic)

			token := client.Publish(prefixedTopic, cfg.QOS, cfg.Retained, payload)
			if err := awaitToken(ctx, token); err != nil {
				l.LogError(ctx, "Failed to publish message to MQTT.", logwrap.Datum("topic", prefixedTopic), logwrap.Err(err))
				return err
			}

			return nil
		}); err != nil {
			l.LogError(context.Background(), "Failed to execute connection handler in MQTT interface.", logwrap.Err(err))
		}
	}

	clientOptions.SetConnectionLostHandler(func(client pahomqtt.Client, err error) {
		l.LogInfo(context.Background(), "MQTT client disconnected.", logwrap.Datum("clientId", clientId), logwrap.Datum("server", cfg.Server), logwrap.Err(err))
		i.Disconnected()
	})

	clientOptions.SetWill(lastWillTopic, `false`, cfg.QOS, cfg.Retained)

	if cfg.Credentials != nil {
		clientOptions.SetUsername(cfg.Credentials.Username)
		clientOptions.SetPassword(cfg.Credentials.Password)
	}

	if cfg.TLS != nil {
		tlsConfig := &tls.Config{InsecureSkipVerify: cfg.TLS.SkipCertificateVerification}

		if cfg.TLS.SkipCertificateVerification {
			l.LogWarn(context.Background(), "Set to ignore remote TLS certificate, this is considered insecure.")
		}

		if len(cfg.TLS.Cert) > 0 {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.Cert, cfg.TLS.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate/key for mqtt: %w", err)
			}

			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		var certPool *x509.CertPool

		if cfg.TLS.IgnoreSystemRootCertificates {
			l.LogInfo(context.Background(), "Configured to ignore system root certificates, ensure you are providing your own.", logwrap.Err(err))
			certPool = x509.NewCertPool()
		} else {
			certPool, err = x509.SystemCertPool()
			if err != nil {
				// This call fails on Windows with an error, but is not typed appropriately so it's impossible to switch, as
				// such we continue on with an empty certificate pool.

				if runtime.GOOS == "windows" {
					l.LogWarn(context.Background(), "Failed to load system certificate pool for root CAs, this is expected on Windows (see Go Issues 16736 and 18609), you must provide the CA root certificate for your servers trust chain.", logwrap.Err(err))
					certPool = x509.NewCertPool()
				} else {
					l.LogError(context.Background(), "Failed to load system certificate pool for root CAs, you may disable loading system certificates by setting $.Config.TLS.IgnoreSystemRootCertificates and provide your own CA certificate.", logwrap.Err(err))
					return nil, fmt.Errorf("failed to load system certiticate pool: %w", err)
				}
			}
		}

		if len(cfg.TLS.CACert) > 0 {
			caCerts, err := ioutil.ReadFile(filepath.Clean(cfg.TLS.CACert))
			if err != nil {
				return nil, fmt.Errorf("failed to load CA TLS certificats for mqtt: %w", err)
			}

			certPool.AppendCertsFromPEM(caCerts)
		}

		tlsConfig.RootCAs = certPool

		clientOptions.SetTLSConfig(tlsConfig)
	}

	i.Start()

	client := pahomqtt.NewClient(clientOptions)

	go func() {
		ctx := context.Background()

		retry := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-retry.C:
				if token := client.Connect(); token.Wait() && token.Error() != nil {
					l.LogError(ctx, "Failed initial connection to MQTT server.", logwrap.Datum("clientId", clientId), logwrap.Datum("server", cfg.Server), logwrap.Err(token.Error()))
				} else {
					l.LogInfo(ctx, "Initial MQTT connection call completed.", logwrap.Datum("clientId", clientId), logwrap.Datum("server", cfg.Server))
					retry.Stop()
					return
				}
			}
		}
	}()

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
