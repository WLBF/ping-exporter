/*
 * Copyright 2022 lbf1353@live.com
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package collector

import (
	"strings"
	"sync"

	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"
)

func newDesc(name, help string, variableLabels []string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc("ping_"+name, help, variableLabels, constLabels)
}

var (
	labelNames   = []string{"src_pod", "src_node", "src_pod_ip", "src_host_ip", "dst_pod", "dst_node", "dst_pod_ip", "dst_host_ip"}
	bestMetric   = newScaledMetric("rtt_best", "Best round trip time", labelNames)
	worstMetric  = newScaledMetric("rtt_worst", "Worst round trip time", labelNames)
	meanMetric   = newScaledMetric("rtt_mean", "Mean round trip time", labelNames)
	stdDevMetric = newScaledMetric("rtt_std_deviation", "Standard deviation", labelNames)
	lossDesc     = newDesc("loss_percent", "Packet loss in percent", labelNames, nil)
	progDesc     = newDesc("up", "ping-exporter version", nil, nil)
	mutex        = &sync.Mutex{}
)

type PingCollector struct {
	Monitor *mon.Monitor
	Metrics map[string]*mon.Metrics
}

func (c *PingCollector) Describe(ch chan<- *prometheus.Desc) {
	bestMetric.Describe(ch)
	worstMetric.Describe(ch)
	meanMetric.Describe(ch)
	stdDevMetric.Describe(ch)
	ch <- lossDesc
	ch <- progDesc
}

func (c *PingCollector) Collect(ch chan<- prometheus.Metric) {
	mutex.Lock()
	defer mutex.Unlock()

	if m := c.Monitor.Export(); len(m) > 0 {
		c.Metrics = m
	}

	for target, metrics := range c.Metrics {
		l := strings.SplitN(target, " ", 8)
		if metrics.PacketsSent > metrics.PacketsLost {
			bestMetric.Collect(ch, metrics.Best, l...)
			worstMetric.Collect(ch, metrics.Worst, l...)
			meanMetric.Collect(ch, metrics.Mean, l...)
			stdDevMetric.Collect(ch, metrics.StdDev, l...)
		}

		loss := float64(metrics.PacketsLost) / float64(metrics.PacketsSent)
		ch <- prometheus.MustNewConstMetric(lossDesc, prometheus.GaugeValue, loss, l...)
	}
}
