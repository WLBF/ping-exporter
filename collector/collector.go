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

import "github.com/prometheus/client_golang/prometheus"

type scaledMetric struct {
	Seconds *prometheus.Desc
}

func (s *scaledMetric) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.Seconds
}

func (s *scaledMetric) Collect(ch chan<- prometheus.Metric, value float32, labelValues ...string) {
	ch <- prometheus.MustNewConstMetric(s.Seconds, prometheus.GaugeValue, float64(value)/1000, labelValues...)
}

func newScaledMetric(name, help string, variableLabels []string) *scaledMetric {
	return &scaledMetric{
		Seconds: newDesc(name+"_seconds", help+" in seconds", variableLabels, nil),
	}
}
