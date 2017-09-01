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

	terminate "github.com/pulcy/go-terminate"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/binkynet/NetManager/service"
	"github.com/binkynet/NetManager/service/mqtt"
)

const (
	projectName = "BinkyNet Network Manager"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
)

func main() {
	var levelFlag string
	var mqttNetwork string
	var mqttAddress string
	var mqttUserName string
	var mqttPassword string
	var mqttTopicName string

	pflag.StringVarP(&levelFlag, "level", "l", "debug", "Set log level")
	pflag.StringVar(&mqttNetwork, "mqtt-network", "tcp", "Network of MQTT connection")
	pflag.StringVar(&mqttAddress, "mqtt-address", "", "Address of MQTT broker")
	pflag.StringVar(&mqttUserName, "mqtt-username", "", "Username of MQTT broker")
	pflag.StringVar(&mqttPassword, "mqtt-password", "", "Password of MQTT broker")
	pflag.StringVar(&mqttTopicName, "mqtt-topicname", "", "Topic name for MQTT messages")
	pflag.Parse()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	mqttSvc, err := mqtt.NewService(mqtt.Config{
		Network:   mqttNetwork,
		Address:   mqttAddress,
		UserName:  mqttUserName,
		Password:  mqttPassword,
		TopicName: mqttTopicName,
	}, logger)
	if err != nil {
		Exitf("Failed to initialize MQTT connection: %v\n", err)
	}

	svc, err := service.NewService(service.ServiceDependencies{
		Log:         logger,
		MqttService: mqttSvc,
	})
	if err != nil {
		Exitf("Failed to initialize Service: %v\n", err)
	}

	// Prepare to shutdown in a controlled manor
	ctx, cancel := context.WithCancel(context.Background())
	t := terminate.NewTerminator(func(template string, args ...interface{}) {
		logger.Info().Msgf(template, args...)
	}, cancel)
	go t.ListenSignals()

	fmt.Printf("Starting %s (version %s build %s)\n", projectName, projectVersion, projectBuild)
	if err := svc.Run(ctx); err != nil {
		Exitf("Service run failed: %#v", err)
	}
}

// Print the given error message and exit with code 1
func Exitf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(1)
}