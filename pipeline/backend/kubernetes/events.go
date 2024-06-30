// Copyright 2024 Woodpecker Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"fmt"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

const (
	// TODO: 5 seconds is against best practice, k3s didn't work otherwise
	defaultResyncDuration = 5 * time.Second
)

type podEventManager struct {
	client      kubernetes.Interface
	config      *config
	stopChan    chan struct{}
	podInformer cache.SharedIndexInformer
}

func newPodEventManager(client kubernetes.Interface, config *config) *podEventManager {
	return &podEventManager{
		client: client,
		config: config,
	}
}

func (pem *podEventManager) start() error {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(pem.client, defaultResyncDuration, informers.WithNamespace(pem.config.Namespace))
	pem.podInformer = informerFactory.Core().V1().Pods().Informer()

	pem.stopChan = make(chan struct{})
	informerFactory.Start(pem.stopChan)
	return nil
}

func (pem *podEventManager) stop() {
	close(pem.stopChan)
}

func (pem *podEventManager) addPodChangeFunc(podName string, onChangeFunc func(pod *v1.Pod)) (cache.ResourceEventHandlerRegistration, error) {
	adapter := newPodChangeFuncResourceEventHandlerAdapter(podName, onChangeFunc)
	return pem.podInformer.AddEventHandler(adapter)
}

func (pem *podEventManager) removePodChangeHandler(handle cache.ResourceEventHandlerRegistration) error {
	return pem.podInformer.RemoveEventHandler(handle)
}

type podChangeFuncResourceEventHandlerAdapter struct {
	podName      string
	onChangeFunc func(pod *v1.Pod)
}

func newPodChangeFuncResourceEventHandlerAdapter(podName string, onChangeFunc func(pod *v1.Pod)) cache.ResourceEventHandler {
	adapter := podChangeFuncResourceEventHandlerAdapter{
		podName:      podName,
		onChangeFunc: onChangeFunc,
	}
	return cache.FilteringResourceEventHandler{
		FilterFunc: adapter.filter,
		Handler:    adapter,
	}
}

func (a podChangeFuncResourceEventHandlerAdapter) OnAdd(obj interface{}, _ bool) {
	pod, err := a.objToPod(obj)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert object to Pod")
	}
	a.onChangeFunc(pod)
}

func (a podChangeFuncResourceEventHandlerAdapter) OnUpdate(_, newObj interface{}) {
	pod, err := a.objToPod(newObj)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert object to Pod")
	}
	a.onChangeFunc(pod)
}

func (a podChangeFuncResourceEventHandlerAdapter) OnDelete(obj interface{}) {
	pod, err := a.objToPod(obj)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert object to Pod")
	}
	a.onChangeFunc(pod)
}

func (a podChangeFuncResourceEventHandlerAdapter) filter(obj interface{}) bool {
	pod, err := a.objToPod(obj)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert object to Pod")
		return false
	}
	return a.podName == pod.Name
}

func (a podChangeFuncResourceEventHandlerAdapter) objToPod(obj interface{}) (*v1.Pod, error) {
	if obj == nil {
		return nil, nil
	}
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return nil, fmt.Errorf("could not parse pod: %v", obj)
	}
	return pod, nil
}
