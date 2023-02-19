// Copyright 2023 Ewout Prangsma
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author Ewout Prangsma
//

package service

import (
	"fmt"

	"github.com/binkynet/NetManager/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	subSystem = "service"
)

type domainMetrics struct {
	SetActualTotalCounters                  *prometheus.CounterVec
	SetRequestTotalCounters                 *prometheus.CounterVec
	WatchTotalCounter                       prometheus.Counter
	WatchActualMessagesTotalCounters        *prometheus.CounterVec
	WatchActualMessagesFailedTotalCounters  *prometheus.CounterVec
	WatchRequestMessagesTotalCounters       *prometheus.CounterVec
	WatchRequestMessagesFailedTotalCounters *prometheus.CounterVec
}

var (
	// Domain metrics
	clockMetrics    = newDomainMetrics("clock")
	discoverMetrics = newDomainMetrics("discover")
	locMetrics      = newDomainMetrics("loc")
	lwMetrics       = newDomainMetrics("lw")
	outputMetrics   = newDomainMetrics("output")
	powerMetrics    = newDomainMetrics("power")
	sensorMetrics   = newDomainMetrics("sensor")
	switchMetrics   = newDomainMetrics("switch")
)

func newDomainMetrics(domain string) domainMetrics {
	return domainMetrics{
		SetActualTotalCounters:                  metrics.MustRegisterCounterVec(subSystem, domain+"_set_actual_total", fmt.Sprintf("number of SetActual calls on %s domain", domain), "address"),
		SetRequestTotalCounters:                 metrics.MustRegisterCounterVec(subSystem, domain+"_set_request_total", fmt.Sprintf("number of SetRequest calls on %s domain", domain), "address"),
		WatchTotalCounter:                       metrics.MustRegisterCounter(subSystem, domain+"_watch_actual_total", fmt.Sprintf("number of SubActual calls on %s domain", domain)),
		WatchActualMessagesTotalCounters:        metrics.MustRegisterCounterVec(subSystem, domain+"_watch_actual_messages_total", fmt.Sprintf("number of actual messages delivered in Watch calls on %s domain", domain), "address"),
		WatchActualMessagesFailedTotalCounters:  metrics.MustRegisterCounterVec(subSystem, domain+"_watch_actual_messages_failed_total", fmt.Sprintf("number of actual messages not delivered in Watch calls on %s domain", domain), "address"),
		WatchRequestMessagesTotalCounters:       metrics.MustRegisterCounterVec(subSystem, domain+"_watch_request_messages_total", fmt.Sprintf("number of request messages delivered in Watch calls on %s domain", domain), "address"),
		WatchRequestMessagesFailedTotalCounters: metrics.MustRegisterCounterVec(subSystem, domain+"_watch_request_messages_failed_total", fmt.Sprintf("number of request messages not delivered in Watch calls on %s domain", domain), "address"),
	}
}
