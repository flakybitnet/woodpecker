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
	"context"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	cleanStaleResourcesId           = "cleanStaleResources"
	maintenanceTaskStartedMessage   = "maintenance task has been started"
	maintenanceTaskCompletedMessage = "maintenance task has been completed"
)

func (e *kube) cleanStaleResources() {
	retention := e.config.CleanupStaleResourcesRetention
	if retention == 0 {
		return
	}
	log.Debug().Str("task", cleanStaleResourcesId).Str("retention", retention.String()).Msg(maintenanceTaskStartedMessage)
	ctx := context.Background()
	namespace := e.config.Namespace

	podList, err := e.client.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Str("task", cleanStaleResourcesId).Msg("failed to get pods")
	}

	deletedPodCount := 0
	for _, pod := range podList.Items {
		if isPodSucceed(&pod) || isPodFailed(&pod) {
			createdAt := time.Unix(pod.CreationTimestamp.Unix(), 0)
			if time.Since(createdAt) > retention {
				log.Debug().
					Str("name", pod.Name).
					Str("createdAt", createdAt.String()).
					Msg("deleting pod")
				err := e.client.CoreV1().Pods(namespace).Delete(ctx, pod.Name, defaultDeleteOptions)
				if err != nil {
					log.Error().Err(err).Str("task", cleanStaleResourcesId).Msg("failed to delete pod")
					continue
				}
				deletedPodCount++
			}
		}
	}

	pvcList, err := e.client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Str("task", cleanStaleResourcesId).Msg("failed to get pvcs")
	}

	deletedPvcCount := 0
	for _, pvc := range pvcList.Items {
		createdAt := time.Unix(pvc.CreationTimestamp.Unix(), 0)
		if time.Since(createdAt) > retention {
			log.Debug().
				Str("name", pvc.Name).
				Str("createdAt", createdAt.String()).
				Msg("deleting PVC")
			err := e.client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvc.Name, defaultDeleteOptions)
			if err != nil {
				log.Error().Err(err).Str("task", cleanStaleResourcesId).Msg("failed to delete pvc")
				continue
			}
			deletedPvcCount++
		}
	}

	svcList, err := e.client.CoreV1().Services(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Str("task", cleanStaleResourcesId).Msg("failed to get services")
	}

	deletedSvcCount := 0
	for _, svc := range svcList.Items {
		createdAt := time.Unix(svc.CreationTimestamp.Unix(), 0)
		if time.Since(createdAt) > retention {
			log.Debug().
				Str("name", svc.Name).
				Str("createdAt", createdAt.String()).
				Msg("deleting service")
			err := e.client.CoreV1().Services(namespace).Delete(ctx, svc.Name, defaultDeleteOptions)
			if err != nil {
				log.Error().Err(err).Str("task", cleanStaleResourcesId).Msg("failed to delete service")
				continue
			}
			deletedSvcCount++
		}
	}

	log.Info().Str("task", cleanStaleResourcesId).
		Int("deletedPodCount", deletedPodCount).Int("deletedPvcCount", deletedPvcCount).Int("deletedSvcCount", deletedSvcCount).
		Msg(maintenanceTaskCompletedMessage)
}
