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

package controller

import (
	"errors"
	"fmt"
	mon "github.com/digineo/go-ping/monitor"
	v1 "k8s.io/api/core/v1"
	informers "k8s.io/client-go/informers/core/v1"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"math/rand"
	"net"
	"time"
)

const RandDurationMs = 2000

type Controller struct {
	me         *v1.Pod
	saved      map[string]*v1.Pod
	monitor    *mon.Monitor
	podsLister listers.PodLister
	podsSynced cache.InformerSynced
}

func New(me *v1.Pod, monitor *mon.Monitor, informer informers.PodInformer) *Controller {
	ctl := Controller{
		me:         me,
		saved:      map[string]*v1.Pod{},
		monitor:    monitor,
		podsLister: informer.Lister(),
		podsSynced: informer.Informer().HasSynced,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctl.addFunc,
		UpdateFunc: ctl.updateFunc,
		DeleteFunc: ctl.deleteFunc,
	})

	return &ctl
}

func randDurationMs(max int) time.Duration {
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(max)
	return time.Duration(n) * time.Millisecond
}

func targetName(src, dst *v1.Pod) string {
	return fmt.Sprintf("%s %s %s %s %s %s %s %s",
		src.Name,
		src.Spec.NodeName,
		src.Status.PodIP,
		src.Status.HostIP,
		dst.Name,
		dst.Spec.NodeName,
		dst.Status.PodIP,
		dst.Status.HostIP,
	)
}

func (ctl *Controller) addTarget(dst *v1.Pod) {
	addr, err := net.ResolveIPAddr("ip", dst.Status.PodIP)
	if err != nil {
		klog.ErrorS(err, "resolve pod ip")
		return
	}

	name := targetName(ctl.me, dst)
	err = ctl.monitor.AddTargetDelayed(name, *addr, randDurationMs(RandDurationMs))
	if err != nil {
		klog.ErrorS(err, "monitor add target error", "name", name)
		return
	}
}

func (ctl *Controller) removeTarget(dst *v1.Pod) {
	name := targetName(ctl.me, dst)
	ctl.monitor.RemoveTarget(name)
}

func (ctl *Controller) addFunc(obj interface{}) {
	pod := obj.(*v1.Pod)
	klog.InfoS("add",
		"pod", klog.KObj(pod),
		"version", pod.ResourceVersion,
		"ip", pod.Status.PodIP)

	// ignore pod without ip
	if len(pod.Status.PodIP) == 0 {
		klog.InfoS("call add with pod without ip", "pod", klog.KObj(pod))
		return
	}

	// consume all saved previous pods
	if ctl.me.Name == pod.Name {
		ctl.me = pod
		for _, dst := range ctl.saved {
			ctl.addTarget(dst)
		}
		return
	}

	// me not have ip yet, save pods
	if len(ctl.me.Status.PodIP) == 0 {
		klog.InfoS("save pod", "pod", klog.KObj(pod))
		ctl.saved[pod.Name] = pod
		return
	}

	ctl.saved[pod.Name] = pod
	ctl.addTarget(pod)
}

func (ctl *Controller) updateFunc(old, new interface{}) {
	oldPod := old.(*v1.Pod)
	newPod := new.(*v1.Pod)

	klog.InfoS("update",
		"old", klog.KObj(oldPod),
		"new", klog.KObj(newPod),
		"oldVersion", oldPod.ResourceVersion,
		"newVersion", newPod.ResourceVersion,
		"oldIP", oldPod.Status.PodIP,
		"newIP", newPod.Status.PodIP)

	if oldPod.ResourceVersion >= newPod.ResourceVersion {
		return
	}

	if len(newPod.Status.PodIP) == 0 {
		klog.InfoS("call update with pod without ip", "pod", klog.KObj(newPod))
		return
	}

	// consume all saved previous pods
	if ctl.me.Name == newPod.Name && ctl.me.Status.PodIP != newPod.Status.PodIP {
		for _, dst := range ctl.saved {
			ctl.removeTarget(dst)
		}

		ctl.me = newPod

		for _, dst := range ctl.saved {
			ctl.addTarget(dst)
		}

		return
	}

	// me not have ip yet, save pods
	if len(ctl.me.Status.PodIP) == 0 {
		klog.InfoS("save pod", "pod", klog.KObj(newPod))
		ctl.saved[newPod.Name] = newPod
		return
	}

	if oldPod.Status.PodIP != newPod.Status.PodIP {
		ctl.saved[newPod.Name] = newPod
		ctl.removeTarget(oldPod)
		ctl.addTarget(newPod)
	}
}

func (ctl *Controller) deleteFunc(obj interface{}) {
	pod := obj.(*v1.Pod)
	klog.InfoS("delete",
		"pod", klog.KObj(pod),
		"version", pod.ResourceVersion,
		"ip", pod.Status.PodIP)
	delete(ctl.saved, pod.Name)
	ctl.removeTarget(pod)
}

func (ctl *Controller) Run(stopCh <-chan struct{}) error {
	// wait for the initial synchronization of the local cache.
	if !cache.WaitForCacheSync(stopCh, ctl.podsSynced) {
		return errors.New("failed  to sync")
	}
	return nil
}
