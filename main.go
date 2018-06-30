//    Copyright 2017 Ewout Prangsma
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/pkg/errors"
	terminate "github.com/pulcy/go-terminate"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	discoveryAPI "github.com/binkynet/BinkyNet/discovery"
	"github.com/binkynet/NetManager/service"
	"github.com/binkynet/NetManager/service/config"
	"github.com/binkynet/NetManager/service/discovery"
	"github.com/binkynet/NetManager/service/server"
)

const (
	projectName       = "BinkyNet Network Manager"
	defaultServerPort = 8823
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
)

func main() {
	var levelFlag string
	var discoveryPort int
	var mqttHost string
	var mqttPort int
	var mqttUserName string
	var mqttPassword string
	var mqttTopicPrefix string
	var registryFolder string
	var serverHost string
	var serverPort int
	var serverEndpoint string

	pflag.StringVarP(&levelFlag, "level", "l", "debug", "Set log level")
	pflag.IntVar(&discoveryPort, "discovery-port", discoveryAPI.DefaultPort, "UDB port used by discovery service")
	pflag.StringVar(&mqttHost, "mqtt-host", "", "Hostname of MQTT connection")
	pflag.IntVar(&mqttPort, "mqtt-port", 1883, "Port of MQTT connection")
	pflag.StringVar(&mqttUserName, "mqtt-username", "", "Username of MQTT broker")
	pflag.StringVar(&mqttPassword, "mqtt-password", "", "Password of MQTT broker")
	pflag.StringVar(&mqttTopicPrefix, "mqtt-topicprefix", "", "Topic prefix for MQTT messages")
	pflag.StringVar(&registryFolder, "folder", "./examples", "Folder containing worker configurations")
	pflag.StringVar(&serverHost, "host", "0.0.0.0", "Host the server is listening on")
	pflag.IntVar(&serverPort, "port", defaultServerPort, "Port the server is listening on")
	pflag.StringVar(&serverEndpoint, "endpoint", "", "Endpoint used to reach the server")
	pflag.Parse()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	if mqttHost == "" {
		Exitf("--mqtt-host missing")
	}
	if serverEndpoint == "" {
		Exitf("--endpoint missing")
	}
	serverEndpoint = mustFixEndpointPort(serverEndpoint, serverPort)
	logger.Debug().
		Str("endpoint", serverEndpoint).
		Msg("Cleaned output")
	origMqttHost := mqttHost
	mqttHost = mustResolveDNSName(mqttHost)
	logger.Debug().
		Str("host", origMqttHost).
		Str("address", mqttHost).
		Msg("Resolved mqtt-host")

	// Prepare to shutdown in a controlled manor
	ctx, cancel := context.WithCancel(context.Background())
	t := terminate.NewTerminator(func(template string, args ...interface{}) {
		logger.Info().Msgf(template, args...)
	}, cancel)
	go t.ListenSignals()

	discoveryMsgs := make(chan discovery.RegisterWorkerMessage)
	discoverySvc, err := discovery.NewService(discovery.Config{
		Port: discoveryPort,
	}, discovery.Dependencies{
		Log:      logger,
		Messages: discoveryMsgs,
	})
	if err != nil {
		Exitf("Failed to initialize discovery service: %v\n", err)
	}

	configReg, err := config.NewRegistry(ctx, registryFolder, mqttTopicPrefix)
	if err != nil {
		Exitf("Failed to initialize worker configuration registry: %v\n", err)
	}

	svc, err := service.NewService(service.Config{
		RequiredWorkerVersion: "",
		MQTTHost:              mqttHost,
		MQTTPort:              mqttPort,
		MQTTUserName:          mqttUserName,
		MQTTPassword:          mqttPassword,
		MyEndpoint:            serverEndpoint,
	}, service.Dependencies{
		Log:               logger,
		ConfigRegistry:    configReg,
		DiscoveryMessages: discoveryMsgs,
	})
	if err != nil {
		Exitf("Failed to initialize Service: %v\n", err)
	}

	httpServer, err := server.NewServer(server.Config{
		Host: serverHost,
		Port: serverPort,
	}, svc, logger)
	if err != nil {
		Exitf("Failed to initialize HTTP server: %v\n", err)
	}

	fmt.Printf("Starting %s (version %s build %s)\n", projectName, projectVersion, projectBuild)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return discoverySvc.Run(ctx) })
	g.Go(func() error { return svc.Run(ctx) })
	g.Go(func() error { return httpServer.Run(ctx) })
	if err := g.Wait(); err != nil && errors.Cause(err) != context.Canceled {
		Exitf("Failed to run services: %#v\n", err)
	}
}

// Exitf prints the given error message and exit with code 1
func Exitf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(1)
}

func mustFixEndpointPort(serverEndpoint string, port int) string {
	u, err := url.Parse(serverEndpoint)
	if err != nil {
		Exitf("Server endpoint is invalid: %s\n", err)
	}
	u.Host = net.JoinHostPort(u.Hostname(), strconv.Itoa(port))
	return u.String()
}

// mustResolveDNSName resolves the given hostname in an IP address.
func mustResolveDNSName(host string) string {
	if ip := net.ParseIP(host); ip != nil {
		// Already IP address
		return host
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		Exitf("Failed to lookup host '%s': %v\n", host, err)
	}
	if len(addrs) == 0 {
		Exitf("Lookup of host '%s' yielded no addresses\n", host)
	}
	return addrs[0]
}
