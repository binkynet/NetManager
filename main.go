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
	"os"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/pkg/errors"
	terminate "github.com/pulcy/go-terminate"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"github.com/binkynet/NetManager/service"
	"github.com/binkynet/NetManager/service/config"
	"github.com/binkynet/NetManager/service/server"
)

const (
	projectName     = "BinkyNet Network Manager"
	defaultGrpcPort = 8823
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
)

func main() {
	var levelFlag string
	var registryFolder string
	var serverHost string
	var grpcPort int

	pflag.StringVarP(&levelFlag, "level", "l", "debug", "Set log level")
	pflag.StringVar(&registryFolder, "folder", "./examples", "Folder containing worker configurations")
	pflag.StringVar(&serverHost, "host", "0.0.0.0", "Host the server is listening on")
	pflag.IntVar(&grpcPort, "port", defaultGrpcPort, "Port the server is listening on")
	pflag.Parse()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	// Prepare to shutdown in a controlled manor
	ctx, cancel := context.WithCancel(context.Background())
	t := terminate.NewTerminator(func(template string, args ...interface{}) {
		logger.Info().Msgf(template, args...)
	}, cancel)
	go t.ListenSignals()

	reconfigureQueue := make(chan string, 64)
	configReg, err := config.NewRegistry(ctx, registryFolder, reconfigureQueue)
	if err != nil {
		Exitf("Failed to initialize worker configuration registry: %v\n", err)
	}

	svc, err := service.NewService(service.Config{
		RequiredWorkerVersion: "",
	}, service.Dependencies{
		Log:              logger,
		ConfigRegistry:   configReg,
		ReconfigureQueue: reconfigureQueue,
	})
	if err != nil {
		Exitf("Failed to initialize Service: %v\n", err)
	}

	server, err := server.NewServer(server.Config{
		Host:     serverHost,
		GRPCPort: grpcPort,
	}, svc, logger)
	if err != nil {
		Exitf("Failed to initialize Server: %v\n", err)
	}

	fmt.Printf("Starting %s (version %s build %s)\n", projectName, projectVersion, projectBuild)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return svc.Run(ctx) })
	g.Go(func() error { return server.Run(ctx) })
	g.Go(func() error {
		return api.RegisterServiceEntry(ctx, api.ServiceTypeLocalWorkerConfig, api.ServiceInfo{
			ApiVersion: "v1",
			ApiPort:    int32(grpcPort),
			Secure:     false,
		})
	})
	g.Go(func() error {
		return api.RegisterServiceEntry(ctx, api.ServiceTypeLocalWorkerControl, api.ServiceInfo{
			ApiVersion: "v1",
			ApiPort:    int32(grpcPort),
			Secure:     false,
		})
	})
	if err := g.Wait(); err != nil && errors.Cause(err) != context.Canceled {
		Exitf("Failed to run services: %#v\n", err)
	}
}

// Exitf prints the given error message and exit with code 1
func Exitf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(1)
}
