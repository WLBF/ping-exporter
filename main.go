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

package main

import (
	"context"
	"fmt"
	"github.com/WLBF/ping-exporter/client"
	"github.com/WLBF/ping-exporter/collector"
	"github.com/WLBF/ping-exporter/controller"
	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"os"
	"time"
)

func NewMonitor() (*mon.Monitor, error) {
	var bind4, bind6 string
	if ln, err := net.Listen("tcp4", "127.0.0.1:0"); err == nil {
		// ipv4 enabled
		ln.Close()
		bind4 = "0.0.0.0"
	}

	if ln, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		// ipv6 enabled
		ln.Close()
		bind6 = "::"
	}

	pinger, err := ping.New(bind4, bind6)
	if err != nil {
		return nil, fmt.Errorf("cannot start monitoring: %w", err)
	}

	monitor := mon.New(pinger, 2*time.Second, 3*time.Second)
	monitor.HistorySize = 128
	return monitor, nil
}

func main() {
	klog.InitFlags(nil)
	podName, ok := os.LookupEnv("MY_POD_NAME")
	if !ok {
		klog.Fatalf("missing env var MY_POD_NAME")
	}
	namespace, ok := os.LookupEnv("MY_POD_NAMESPACE")
	if !ok {
		klog.Fatalf("missing env var MY_POD_NAMESPACE")
	}
	klog.Infoln("ping-exporter", "namespace", namespace, "pod", podName)

	var cli *kubernetes.Clientset

	cli, err := client.InClusterClient()
	if err != nil {
		klog.ErrorS(err, "cannot create in-cluster client")
		kubeConfig, _ := os.LookupEnv("KUBECONFIG")
		cli, err = client.NewClientFromKubeConfig(kubeConfig)
		if err != nil {
			klog.Fatal("cannot create out of cluster client", err)
		}
	}

	opt := v1.ListOptions{
		FieldSelector: "metadata.name=" + podName,
	}
	podList, err := cli.CoreV1().Pods(namespace).List(context.Background(), opt)
	if err != nil {
		klog.Fatal("get pod", err)
	}
	if len(podList.Items) != 1 {
		klog.Fatal("get pod", "expected 1 pod, got", len(podList.Items))
	}

	pod := podList.Items[0]
	if len(pod.OwnerReferences) != 1 {
		klog.Fatal("no owner reference pod")
	}

	ownerRef := pod.OwnerReferences[0]
	if ownerRef.Kind != "DaemonSet" {
		klog.Fatalf("owner ref %s not DaemonSet", ownerRef.Kind)
	}

	ownerName := ownerRef.Name
	opt = v1.ListOptions{
		FieldSelector: "metadata.name=" + ownerName,
	}

	daemonSetList, err := cli.AppsV1().DaemonSets(namespace).List(context.Background(), opt)
	matchLabels := daemonSetList.Items[0].Spec.Selector.MatchLabels

	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(cli, time.Second*30, informers.WithNamespace(namespace), informers.WithTweakListOptions(func(options *v1.ListOptions) {
		options.LabelSelector = labels.SelectorFromSet(matchLabels).String()
	}))

	podsInformer := kubeInformerFactory.Core().V1().Pods()

	monitor, err := NewMonitor()
	if err != nil {
		klog.Fatal("cannot create monitor", err)
	}
	op := controller.New(&pod, monitor, podsInformer)
	stop := make(chan struct{})
	defer close(stop)
	kubeInformerFactory.Start(stop)
	op.Run(stop)
	prometheus.MustRegister(&collector.PingCollector{Monitor: monitor})
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}
