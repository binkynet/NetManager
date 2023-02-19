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

package manager

import (
	"fmt"

	"github.com/binkynet/NetManager/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	subSystem = "manager"
)

type poolMetrics struct {
	SetActualTotalCounters                *prometheus.CounterVec
	SetRequestTotalCounters               *prometheus.CounterVec
	SubActualTotalCounter                 prometheus.Counter
	SubRequestTotalCounter                prometheus.Counter
	SubActualMessagesTotalCounters        *prometheus.CounterVec
	SubActualMessagesFailedTotalCounters  *prometheus.CounterVec
	SubRequestMessagesTotalCounters       *prometheus.CounterVec
	SubRequestMessagesFailedTotalCounters *prometheus.CounterVec
}

var (
	// Pool metrics
	clockPoolMetrics    = newPoolMetrics("clock")
	discoverPoolMetrics = newPoolMetrics("discover")
	locPoolMetrics      = newPoolMetrics("loc")
	lwPoolMetrics       = newPoolMetrics("lw")
	outputPoolMetrics   = newPoolMetrics("output")
	powerPoolMetrics    = newPoolMetrics("power")
	sensorPoolMetrics   = newPoolMetrics("sensor")
	switchPoolMetrics   = newPoolMetrics("switch")
)

func newPoolMetrics(pool string) poolMetrics {
	return poolMetrics{
		SetActualTotalCounters:                metrics.MustRegisterCounterVec(subSystem, pool+"_pool_set_actual_total", fmt.Sprintf("number of SetActual calls on %s pool", pool), "address"),
		SetRequestTotalCounters:               metrics.MustRegisterCounterVec(subSystem, pool+"_pool_set_request_total", fmt.Sprintf("number of SetRequest calls on %s pool", pool), "address"),
		SubActualTotalCounter:                 metrics.MustRegisterCounter(subSystem, pool+"_pool_sub_actual_total", fmt.Sprintf("number of SubActual calls on %s pool", pool)),
		SubRequestTotalCounter:                metrics.MustRegisterCounter(subSystem, pool+"_pool_sub_request_total", fmt.Sprintf("number of SubRequest calls on %s pool", pool)),
		SubActualMessagesTotalCounters:        metrics.MustRegisterCounterVec(subSystem, pool+"_pool_sub_actual_messages_total", fmt.Sprintf("number of messages delivered in SubActual calls on %s pool", pool), "address"),
		SubActualMessagesFailedTotalCounters:  metrics.MustRegisterCounterVec(subSystem, pool+"_pool_sub_actual_messages_failed_total", fmt.Sprintf("number of messages not delivered in SubActual calls on %s pool", pool), "address"),
		SubRequestMessagesTotalCounters:       metrics.MustRegisterCounterVec(subSystem, pool+"_pool_sub_request_messages_total", fmt.Sprintf("number of messages delivered in SubRequest calls on %s pool", pool), "address"),
		SubRequestMessagesFailedTotalCounters: metrics.MustRegisterCounterVec(subSystem, pool+"_pool_sub_request_messages_failed_total", fmt.Sprintf("number of messages not delivered in SubRequest calls on %s pool", pool), "address"),
	}
}
