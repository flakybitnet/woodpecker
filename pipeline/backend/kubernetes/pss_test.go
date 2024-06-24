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
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPssEnabled(t *testing.T) {
	pssp := newPssProcessor(&config{PssProfile: "restricted"})
	assert.Equal(t, true, pssp.isEnabled())

	pssp = newPssProcessor(&config{PssProfile: "baseline"})
	assert.Equal(t, true, pssp.isEnabled())
}

func TestPssDisabled(t *testing.T) {
	pssp := newPssProcessor(&config{})
	assert.Equal(t, false, pssp.isEnabled())

	pssp = newPssProcessor(&config{PssProfile: ""})
	assert.Equal(t, false, pssp.isEnabled())

	pssp = newPssProcessor(&config{PssProfile: "other"})
	assert.Equal(t, false, pssp.isEnabled())

	pssp = newPssProcessor(&config{PssProfile: "privileged"})
	assert.Equal(t, false, pssp.isEnabled())
}

func createTestPssPod(podSecCtx *v1.PodSecurityContext, containerSecCtx *v1.SecurityContext) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "wp-01he8begk0qj36dbctabr3k2me-0",
		},
		Spec: v1.PodSpec{
			SecurityContext: podSecCtx,
			Containers: []v1.Container{
				{
					Name:            "step-0",
					Image:           "alpine",
					SecurityContext: containerSecCtx,
				},
			},
		},
	}
}

func TestBasePssEmptyPod(t *testing.T) {
	pssp := newPssProcessor(&config{
		PssProfile: PssProfileBaseline,
	})

	pod := createTestPssPod(nil, nil)
	pssp.apply(pod)

	podSecCtx := pod.Spec.SecurityContext
	assert.NotNil(t, podSecCtx)
	assert.Equal(t, &v1.PodSecurityContext{}, podSecCtx)

	containerSecCtx := pod.Spec.Containers[0].SecurityContext
	assert.NotNil(t, containerSecCtx)
	assert.Equal(t, &v1.SecurityContext{}, containerSecCtx)
}

func TestBasePssPod(t *testing.T) {
	pssp := newPssProcessor(&config{
		PssProfile: PssProfileBaseline,
	})

	pod := createTestPssPod(
		&v1.PodSecurityContext{
			RunAsNonRoot: newBool(false),
			RunAsUser:    newInt64(0),
			RunAsGroup:   newInt64(0),
			SeccompProfile: &v1.SeccompProfile{
				Type: v1.SeccompProfileTypeUnconfined,
			},
		},
		&v1.SecurityContext{
			Privileged:   newBool(true),
			RunAsNonRoot: newBool(false),
			RunAsUser:    newInt64(0),
			RunAsGroup:   newInt64(0),
			SeccompProfile: &v1.SeccompProfile{
				Type: v1.SeccompProfileTypeUnconfined,
			},
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{CapabilityAll},
			},
		})
	pssp.apply(pod)

	podSecCtx := pod.Spec.SecurityContext
	assert.NotNil(t, podSecCtx)
	assert.Equal(t, newBool(false), podSecCtx.RunAsNonRoot)
	assert.Equal(t, newInt64(0), podSecCtx.RunAsUser)
	assert.Equal(t, newInt64(0), podSecCtx.RunAsGroup)
	assert.NotNil(t, podSecCtx.SeccompProfile)
	assert.Equal(t, v1.SeccompProfileTypeRuntimeDefault, podSecCtx.SeccompProfile.Type)

	containerSecCtx := pod.Spec.Containers[0].SecurityContext
	assert.NotNil(t, containerSecCtx)
	assert.Equal(t, newBool(false), containerSecCtx.Privileged)
	assert.Equal(t, newBool(false), containerSecCtx.RunAsNonRoot)
	assert.Equal(t, newInt64(0), containerSecCtx.RunAsUser)
	assert.Equal(t, newInt64(0), containerSecCtx.RunAsGroup)
	assert.NotNil(t, containerSecCtx.SeccompProfile)
	assert.Equal(t, v1.SeccompProfileTypeRuntimeDefault, containerSecCtx.SeccompProfile.Type)
	assert.NotNil(t, containerSecCtx.Capabilities)
	assert.Empty(t, containerSecCtx.Capabilities.Add)
}

func TestStrictPssEmptyPod(t *testing.T) {
	pssp := newPssProcessor(&config{
		PssProfile: PssProfileRestricted,
	})

	pod := createTestPssPod(nil, nil)
	pssp.apply(pod)

	podSecCtx := pod.Spec.SecurityContext
	assert.NotNil(t, podSecCtx)
	assert.Equal(t, newBool(true), podSecCtx.RunAsNonRoot)
	assert.Nil(t, podSecCtx.RunAsUser)
	assert.Nil(t, podSecCtx.RunAsGroup)
	assert.NotNil(t, podSecCtx.SeccompProfile)
	assert.Equal(t, v1.SeccompProfileTypeRuntimeDefault, podSecCtx.SeccompProfile.Type)

	containerSecCtx := pod.Spec.Containers[0].SecurityContext
	assert.NotNil(t, containerSecCtx)
	assert.Nil(t, containerSecCtx.Privileged)
	assert.Equal(t, newBool(false), containerSecCtx.AllowPrivilegeEscalation)
	assert.Nil(t, containerSecCtx.RunAsNonRoot)
	assert.Nil(t, containerSecCtx.RunAsUser)
	assert.Nil(t, containerSecCtx.RunAsGroup)
	assert.Nil(t, containerSecCtx.SeccompProfile)
	assert.NotNil(t, containerSecCtx.Capabilities)
	assert.Empty(t, containerSecCtx.Capabilities.Add)
	assert.Equal(t, []v1.Capability{CapabilityAll}, containerSecCtx.Capabilities.Drop)
}

func TestStrictPssPod(t *testing.T) {
	pssp := newPssProcessor(&config{
		PssProfile: PssProfileRestricted,
	})

	pod := createTestPssPod(
		&v1.PodSecurityContext{
			RunAsNonRoot: newBool(false),
			RunAsUser:    newInt64(0),
			RunAsGroup:   newInt64(0),
			SeccompProfile: &v1.SeccompProfile{
				Type: v1.SeccompProfileTypeUnconfined,
			},
		},
		&v1.SecurityContext{
			Privileged:   newBool(true),
			RunAsNonRoot: newBool(false),
			RunAsUser:    newInt64(0),
			RunAsGroup:   newInt64(0),
			SeccompProfile: &v1.SeccompProfile{
				Type: v1.SeccompProfileTypeUnconfined,
			},
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{CapabilityKill},
			},
		})
	pssp.apply(pod)

	podSecCtx := pod.Spec.SecurityContext
	assert.NotNil(t, podSecCtx)
	assert.Equal(t, newBool(true), podSecCtx.RunAsNonRoot)
	assert.Nil(t, podSecCtx.RunAsUser)
	assert.Equal(t, newInt64(0), podSecCtx.RunAsGroup)
	assert.NotNil(t, podSecCtx.SeccompProfile)
	assert.Equal(t, v1.SeccompProfileTypeRuntimeDefault, podSecCtx.SeccompProfile.Type)

	containerSecCtx := pod.Spec.Containers[0].SecurityContext
	assert.NotNil(t, containerSecCtx)
	assert.Equal(t, newBool(false), containerSecCtx.Privileged)
	assert.Equal(t, newBool(false), containerSecCtx.AllowPrivilegeEscalation)
	assert.Nil(t, containerSecCtx.RunAsNonRoot)
	assert.Nil(t, containerSecCtx.RunAsUser)
	assert.Equal(t, newInt64(0), containerSecCtx.RunAsGroup)
	assert.NotNil(t, containerSecCtx.SeccompProfile)
	assert.Equal(t, v1.SeccompProfileTypeRuntimeDefault, containerSecCtx.SeccompProfile.Type)
	assert.NotNil(t, containerSecCtx.Capabilities)
	assert.Empty(t, containerSecCtx.Capabilities.Add)
	assert.Equal(t, []v1.Capability{CapabilityAll}, containerSecCtx.Capabilities.Drop)
}
