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
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	CapabilityAll            v1.Capability = "ALL"
	CapabilityChown          v1.Capability = "CHOWN"
	CapabilityDacOverride    v1.Capability = "DAC_OVERRIDE"
	CapabilityFowner         v1.Capability = "FOWNER"
	CapabilityFsetid         v1.Capability = "FSETID"
	CapabilityKill           v1.Capability = "KILL"
	CapabilityMknod          v1.Capability = "MKNOD"
	CapabilityNetBindService v1.Capability = "NET_BIND_SERVICE"
	CapabilitySetfcap        v1.Capability = "SETFCAP"
	CapabilitySetgid         v1.Capability = "SETGID"
	CapabilitySetpcap        v1.Capability = "SETPCAP"
	CapabilitySetuid         v1.Capability = "SETUID"
	CapabilitySysChroot      v1.Capability = "SYS_CHROOT"
)

var (
	BaseCapabilities = sets.New(
		CapabilityChown,
		CapabilityDacOverride,
		CapabilityFowner,
		CapabilityFsetid,
		CapabilityKill,
		CapabilityMknod,
		CapabilityNetBindService,
		CapabilitySetfcap,
		CapabilitySetgid,
		CapabilitySetpcap,
		CapabilitySetuid,
		CapabilitySysChroot,
	)
	StrictCapabilities = sets.New(
		CapabilityNetBindService,
	)
)

type pssProcessor struct {
	profile PssProfile
}

func newPssProcessor(config *config) *pssProcessor {
	return &pssProcessor{
		profile: config.PssProfile,
	}
}

func (p *pssProcessor) isEnabled() bool {
	return p.profile == PssProfileBaseline || p.profile == PssProfileRestricted
}

func (p *pssProcessor) apply(pod *v1.Pod) {
	if !p.isEnabled() {
		return
	}

	p.initPodSecCtx(pod)
	p.initContainersSecCtx(pod)

	switch p.profile {
	case PssProfileBaseline:
		p.applyBaseline(pod)
	case PssProfileRestricted:
		p.applyRestricted(pod)
	}
}

func (p *pssProcessor) applyBaseline(pod *v1.Pod) {
	log.Trace().Msg("applying baseline pod security standard")
	p.baseDisallowPrivilegedContainers(pod)
	p.baseSeccomp(pod)
	p.baseCapabilities(pod)
}

func (p *pssProcessor) applyRestricted(pod *v1.Pod) {
	log.Trace().Msg("applying restricted pod security standard")
	p.baseDisallowPrivilegedContainers(pod)
	p.strictDisallowPrivilegeEscalation(pod)
	p.strictRunAsNonRoot(pod)
	p.strictSeccomp(pod)
	p.strictCapabilities(pod)
}

func (p *pssProcessor) initPodSecCtx(pod *v1.Pod) {
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &v1.PodSecurityContext{}
	}
}

func (p *pssProcessor) initContainersSecCtx(pod *v1.Pod) {
	containers := pod.Spec.Containers
	for i, _ := range containers {
		if containers[i].SecurityContext == nil {
			containers[i].SecurityContext = &v1.SecurityContext{}
		}
	}
}

func (p *pssProcessor) baseDisallowPrivilegedContainers(pod *v1.Pod) {
	containers := pod.Spec.Containers
	for i, _ := range containers {
		secCtx := containers[i].SecurityContext
		if secCtx.Privileged != nil && *secCtx.Privileged == true {
			secCtx.Privileged = newBool(false)
		}
	}
}

func (p *pssProcessor) baseSeccomp(pod *v1.Pod) {
	podSecCtx := pod.Spec.SecurityContext
	if podSecCtx.SeccompProfile != nil {
		if podSecCtx.SeccompProfile.Type == v1.SeccompProfileTypeUnconfined {
			podSecCtx.SeccompProfile.Type = v1.SeccompProfileTypeRuntimeDefault
		}
	}

	containers := pod.Spec.Containers
	for i, _ := range containers {
		containerSecCtx := containers[i].SecurityContext
		if containerSecCtx.SeccompProfile != nil {
			if containerSecCtx.SeccompProfile.Type == v1.SeccompProfileTypeUnconfined {
				containerSecCtx.SeccompProfile.Type = v1.SeccompProfileTypeRuntimeDefault
			}
		}
	}
}

func (p *pssProcessor) baseCapabilities(pod *v1.Pod) {
	containers := pod.Spec.Containers
	for i, _ := range containers {
		containerSecCtx := containers[i].SecurityContext
		if containerSecCtx.Capabilities != nil {
			if len(containerSecCtx.Capabilities.Add) > 0 {
				acaps := sets.New(containerSecCtx.Capabilities.Add...)
				allowedCaps := acaps.Intersection(BaseCapabilities)
				containerSecCtx.Capabilities.Add = allowedCaps.UnsortedList()
			}
		}
	}
}

func (p *pssProcessor) strictDisallowPrivilegeEscalation(pod *v1.Pod) {
	containers := pod.Spec.Containers
	for i, _ := range containers {
		secCtx := containers[i].SecurityContext
		secCtx.AllowPrivilegeEscalation = newBool(false)
	}
}

func (p *pssProcessor) strictRunAsNonRoot(pod *v1.Pod) {
	podSecCtx := pod.Spec.SecurityContext
	podSecCtx.RunAsNonRoot = newBool(true)
	if podSecCtx.RunAsUser != nil && *podSecCtx.RunAsUser == 0 {
		podSecCtx.RunAsUser = nil
	}

	containers := pod.Spec.Containers
	for i, _ := range containers {
		containerSecCtx := containers[i].SecurityContext
		if containerSecCtx.RunAsNonRoot != nil && *containerSecCtx.RunAsNonRoot == false {
			containerSecCtx.RunAsNonRoot = nil
		}
		if containerSecCtx.RunAsUser != nil && *containerSecCtx.RunAsUser == 0 {
			containerSecCtx.RunAsUser = nil
		}
	}
}

func (p *pssProcessor) strictSeccomp(pod *v1.Pod) {
	podSecCtx := pod.Spec.SecurityContext
	if podSecCtx.SeccompProfile == nil {
		podSecCtx.SeccompProfile = &v1.SeccompProfile{}
	}
	if podSecCtx.SeccompProfile.Type != v1.SeccompProfileTypeRuntimeDefault && podSecCtx.SeccompProfile.Type != v1.SeccompProfileTypeLocalhost {
		podSecCtx.SeccompProfile.Type = v1.SeccompProfileTypeRuntimeDefault
	}

	containers := pod.Spec.Containers
	for i, _ := range containers {
		containerSecCtx := containers[i].SecurityContext
		if containerSecCtx.SeccompProfile != nil {
			if containerSecCtx.SeccompProfile.Type != v1.SeccompProfileTypeRuntimeDefault && containerSecCtx.SeccompProfile.Type != v1.SeccompProfileTypeLocalhost {
				containerSecCtx.SeccompProfile.Type = v1.SeccompProfileTypeRuntimeDefault
			}
		}
	}
}

func (p *pssProcessor) strictCapabilities(pod *v1.Pod) {
	containers := pod.Spec.Containers
	for i, _ := range containers {
		containerSecCtx := containers[i].SecurityContext
		if containerSecCtx.Capabilities == nil {
			containerSecCtx.Capabilities = &v1.Capabilities{}
		}

		containerSecCtx.Capabilities.Drop = []v1.Capability{CapabilityAll}
		if len(containerSecCtx.Capabilities.Add) > 0 {
			acaps := sets.New(containerSecCtx.Capabilities.Add...)
			allowedCaps := acaps.Intersection(StrictCapabilities)
			containerSecCtx.Capabilities.Add = allowedCaps.UnsortedList()
		}
	}
}
