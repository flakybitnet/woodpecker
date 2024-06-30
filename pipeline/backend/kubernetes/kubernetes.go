// Copyright 2022 Woodpecker Authors
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

package kubernetes

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/backend/types"
	"gopkg.in/yaml.v3"
	"io"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // To authenticate to GCP K8s clusters
	"k8s.io/client-go/rest"
	"maps"
	"os"
	"runtime"
	"slices"
)

const (
	EngineName = "kubernetes"
)

const (
	OomKilledExitCode               = 137
	ContainerReasonImagePullBackOff = "ImagePullBackOff"
	ContainerReasonInvalidImageName = "InvalidImageName"
)

var defaultDeleteOptions = newDefaultDeleteOptions()

type kube struct {
	client          kubernetes.Interface
	config          *config
	goos            string
	pssProc         *pssProcessor
	podEventManager *podEventManager
}

type config struct {
	Namespace                   string
	StorageClass                string
	VolumeSize                  string
	StorageRwx                  bool
	PodLabels                   map[string]string
	PodLabelsAllowFromStep      bool
	PodAnnotations              map[string]string
	PodAnnotationsAllowFromStep bool
	PodNodeSelector             map[string]string
	ImagePullSecretNames        []string
	PodUserHome                 string
	SecurityContext             SecurityContextConfig
	NativeSecretsAllowFromStep  bool
	PssProfile                  PssProfile
}
type SecurityContextConfig struct {
	RunAsNonRoot bool
	User         int64
	Group        int64
	FsGroup      int64
}

type PssProfile string

const (
	PssProfileBaseline   PssProfile = "baseline"
	PssProfileRestricted PssProfile = "restricted"
)

func newDefaultDeleteOptions() meta_v1.DeleteOptions {
	gracePeriodSeconds := int64(0) // immediately
	propagationPolicy := meta_v1.DeletePropagationBackground

	return meta_v1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
		PropagationPolicy:  &propagationPolicy,
	}
}

func configFromCliContext(ctx context.Context) (*config, error) {
	if ctx != nil {
		if c, ok := ctx.Value(types.CliCommand).(*cli.Command); ok {
			config := config{
				Namespace:                   c.String("backend-k8s-namespace"),
				StorageClass:                c.String("backend-k8s-storage-class"),
				VolumeSize:                  c.String("backend-k8s-volume-size"),
				StorageRwx:                  c.Bool("backend-k8s-storage-rwx"),
				PodLabels:                   make(map[string]string), // just init empty map to prevent nil panic
				PodLabelsAllowFromStep:      c.Bool("backend-k8s-pod-labels-allow-from-step"),
				PodAnnotations:              make(map[string]string), // just init empty map to prevent nil panic
				PodAnnotationsAllowFromStep: c.Bool("backend-k8s-pod-annotations-allow-from-step"),
				PodNodeSelector:             make(map[string]string), // just init empty map to prevent nil panic
				ImagePullSecretNames:        c.StringSlice("backend-k8s-pod-image-pull-secret-names"),
				PodUserHome:                 c.String("backend-k8s-pod-user-home"),
				PssProfile:                  PssProfile(c.String("backend-k8s-pss-profile")),
				SecurityContext: SecurityContextConfig{
					RunAsNonRoot: c.Bool("backend-k8s-secctx-nonroot"), // cspell:words secctx nonroot
					User:         c.Int("backend-k8s-secctx-user"),
					Group:        c.Int("backend-k8s-secctx-group"),
					FsGroup:      c.Int("backend-k8s-secctx-fsgroup"),
				},
				NativeSecretsAllowFromStep: c.Bool("backend-k8s-allow-native-secrets"),
			}
			// TODO: remove in next major
			if len(config.ImagePullSecretNames) == 1 && config.ImagePullSecretNames[0] == "regcred" {
				log.Warn().Msg("WOODPECKER_BACKEND_K8S_PULL_SECRET_NAMES is set to the default ('regcred'). It will default to empty in Woodpecker 3.0. Set it explicitly before then.")
			}
			// Unmarshal label and annotation settings here to ensure they're valid on startup
			if labels := c.String("backend-k8s-pod-labels"); labels != "" {
				if err := yaml.Unmarshal([]byte(labels), &config.PodLabels); err != nil {
					log.Error().Err(err).Msgf("could not unmarshal pod labels '%s'", c.String("backend-k8s-pod-labels"))
					return nil, err
				}
			}
			if annotations := c.String("backend-k8s-pod-annotations"); annotations != "" {
				if err := yaml.Unmarshal([]byte(c.String("backend-k8s-pod-annotations")), &config.PodAnnotations); err != nil {
					log.Error().Err(err).Msgf("could not unmarshal pod annotations '%s'", c.String("backend-k8s-pod-annotations"))
					return nil, err
				}
			}
			if nodeSelector := c.String("backend-k8s-pod-node-selector"); nodeSelector != "" {
				if err := yaml.Unmarshal([]byte(nodeSelector), &config.PodNodeSelector); err != nil {
					log.Error().Err(err).Msgf("could not unmarshal pod node selector '%s'", nodeSelector)
					return nil, err
				}
			}
			return &config, nil
		}
	}

	return nil, types.ErrNoCliContextFound
}

// New returns a new Kubernetes Backend.
func New() types.Backend {
	return &kube{}
}

func (e *kube) Name() string {
	return EngineName
}

func (e *kube) IsAvailable(context.Context) bool {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	return len(host) > 0
}

func (e *kube) Flags() []cli.Flag {
	return Flags
}

func (e *kube) Load(ctx context.Context) (*types.BackendInfo, error) {
	config, err := configFromCliContext(ctx)
	if err != nil {
		return nil, err
	}
	e.config = config

	var kubeClient kubernetes.Interface
	_, err = rest.InClusterConfig()
	if err != nil {
		kubeClient, err = getClientOutOfCluster()
	} else {
		kubeClient, err = getClientInsideOfCluster()
	}

	if err != nil {
		return nil, err
	}

	e.client = kubeClient

	e.pssProc = newPssProcessor(e.getConfig())

	e.podEventManager = newPodEventManager(e.client, e.getConfig())
	err = e.podEventManager.start()
	if err != nil {
		return nil, err
	}

	go e.unload(ctx)

	// TODO(2693): use info resp of kubeClient to define platform var
	e.goos = runtime.GOOS
	return &types.BackendInfo{
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}, nil
}

func (e *kube) unload(ctx context.Context) {
	select {
	case <-ctx.Done():
		e.podEventManager.stop()
	}
}

func (e *kube) getConfig() *config {
	if e.config == nil {
		return nil
	}
	c := *e.config
	c.PodLabels = maps.Clone(e.config.PodLabels)
	c.PodAnnotations = maps.Clone(e.config.PodAnnotations)
	c.PodNodeSelector = maps.Clone(e.config.PodNodeSelector)
	c.ImagePullSecretNames = slices.Clone(e.config.ImagePullSecretNames)
	return &c
}

// SetupWorkflow sets up the pipeline environment.
func (e *kube) SetupWorkflow(ctx context.Context, conf *types.Config, taskUUID string) error {
	log.Trace().Str("taskUUID", taskUUID).Msgf("Setting up Kubernetes primitives")

	for _, vol := range conf.Volumes {
		_, err := startVolume(ctx, e, vol.Name)
		if err != nil {
			return err
		}
	}

	var extraHosts []types.HostAlias
	for _, stage := range conf.Stages {
		for _, step := range stage.Steps {
			if step.Type == types.StepTypeService {
				svc, err := startService(ctx, e, step)
				if err != nil {
					return err
				}
				hostAlias := types.HostAlias{Name: step.Networks[0].Aliases[0], IP: svc.Spec.ClusterIP}
				extraHosts = append(extraHosts, hostAlias)
			}
		}
	}
	log.Trace().Msgf("adding extra hosts: %v", extraHosts)
	for _, stage := range conf.Stages {
		for _, step := range stage.Steps {
			step.ExtraHosts = extraHosts
		}
	}

	return nil
}

// StartStep starts the pipeline step.
func (e *kube) StartStep(ctx context.Context, step *types.Step, taskUUID string) error {
	options, err := parseBackendOptions(step)
	if err != nil {
		log.Error().Err(err).Msg("could not parse backend options")
	}

	log.Trace().Str("taskUUID", taskUUID).Msgf("starting step: %s", step.Name)
	pod, err := startPod(ctx, e, step, options)
	if err != nil {
		return err
	}

	pod, err = e.waitStart(ctx, pod.Name)
	if err != nil {
		return err
	}

	switch {
	case isPodPending(pod): // pending unrecoverable
		return newPodPendingError(pod)
	case isPodFailed(pod): // failed
		// if it is failed now, then it was run before =>
		// we should gather the logs and handle it in WaitStep
		return nil
	default: // running or succeed
		return nil
	}
}

func (e *kube) waitStart(ctx context.Context, podName string) (*v1.Pod, error) {
	podStartedOrUnrecoverableChan := make(chan *v1.Pod)
	defer close(podStartedOrUnrecoverableChan)

	var lastSeenPod *v1.Pod
	registration, err := e.podEventManager.addPodChangeFunc(podName, func(pod *v1.Pod) {
		lastSeenPod = pod
		if !isPodPending(pod) {
			podStartedOrUnrecoverableChan <- pod
		} else {
			logPendingPod(pod)
			if isPodPendingUnrecoverable(pod) {
				podStartedOrUnrecoverableChan <- pod
			}
		}
	})
	if err != nil {
		return nil, err
	}
	defer e.podEventManager.removePodChangeHandler(registration)

	for {
		select {
		case <-ctx.Done():
			return lastSeenPod, ctx.Err()
		case startedOrUnrecoverablePod := <-podStartedOrUnrecoverableChan:
			return startedOrUnrecoverablePod, nil
		}
	}
}

func logPendingPod(pod *v1.Pod) {
	ps := pod.Status
	l := log.Trace().Str("pod", pod.Name).Str("podStatus", string(ps.Phase)).Str("podReason", ps.Reason).Str("podMessage", ps.Message)
	cs := getFirstContainerState(pod)
	if cs != nil && cs.Waiting != nil {
		l.Str("containerReason", cs.Waiting.Reason).Str("containerMessage", cs.Waiting.Message)
	}
	l.Msg("pod is pending")
}

func isPodPendingUnrecoverable(pod *v1.Pod) bool {
	if !isPodPending(pod) {
		return false
	}
	cs := getFirstContainerState(pod)
	if cs == nil || cs.Waiting == nil {
		return false
	}
	return cs.Waiting.Reason == ContainerReasonImagePullBackOff || cs.Waiting.Reason == ContainerReasonInvalidImageName
}

// WaitStep waits for the pipeline step to complete and returns
// the completion results.
func (e *kube) WaitStep(ctx context.Context, step *types.Step, taskUUID string) (*types.State, error) {
	log.Trace().Str("taskUUID", taskUUID).Str("step", step.UUID).Msg("waiting for step")
	podName, err := stepToPodName(step)
	if err != nil {
		return nil, err
	}

	pod, err := e.waitStop(ctx, podName)
	if err != nil {
		return nil, err
	}

	state := types.State{
		Exited: true,
	}
	cs := getFirstContainerState(pod)

	switch {
	case isPodSucceed(pod):
		state.ExitCode = 0
	case isPodFailed(pod):
		state.ExitCode = 1
		// if we set error here, then we can't see step logs, there will be something like:
		// Oh no, we got some errors!
		// pod wp-01j1nebq3a06xhmp144bze92xk failed because of , : container containerd://e579b7b385acd1ae1b2ba24144928eb92607cae804cc5ca360a3832a2eeb3186 is terminated because of Error,
		//state.Error = newPodFailedError(pod)
		if cs != nil && cs.Terminated != nil {
			state.ExitCode = int(cs.Terminated.ExitCode)
			state.OOMKilled = cs.Terminated.ExitCode == OomKilledExitCode
		}
	}

	return &state, nil
}

func (e *kube) waitStop(ctx context.Context, podName string) (*v1.Pod, error) {
	podStoppedChan := make(chan *v1.Pod)
	defer close(podStoppedChan)

	var lastSeenPod *v1.Pod
	registration, err := e.podEventManager.addPodChangeFunc(podName, func(pod *v1.Pod) {
		lastSeenPod = pod
		if isPodSucceed(pod) || isPodFailed(pod) {
			podStoppedChan <- pod
		}
	})
	if err != nil {
		return nil, err
	}
	defer e.podEventManager.removePodChangeHandler(registration)

	for {
		select {
		case <-ctx.Done():
			return lastSeenPod, ctx.Err()
		case stoppedPod := <-podStoppedChan:
			return stoppedPod, nil
		}
	}
}

// TailStep tails the pipeline step logs.
func (e *kube) TailStep(ctx context.Context, step *types.Step, taskUUID string) (io.ReadCloser, error) {
	podName, err := stepToPodName(step)
	if err != nil {
		return nil, err
	}
	log.Trace().Str("taskUUID", taskUUID).Msgf("tail logs of pod: %s", podName)

	logsStream, err := e.streamLogs(ctx, step)
	if err != nil {
		return nil, err
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		defer logsStream.Close()
		defer pipeWriter.Close()
		defer pipeReader.Close()

		_, err = io.Copy(pipeWriter, logsStream)
		if err != nil {
			return
		}
	}()
	return pipeReader, nil
}

func (e *kube) streamLogs(ctx context.Context, step *types.Step) (io.ReadCloser, error) {
	podName, err := stepToPodName(step)
	if err != nil {
		return nil, err
	}
	opts := &v1.PodLogOptions{
		Follow:    true,
		Container: podName,
	}
	return e.client.CoreV1().RESTClient().Get().
		Namespace(e.config.Namespace).
		Name(podName).
		Resource("pods").
		SubResource("log").
		VersionedParams(opts, scheme.ParameterCodec).
		Stream(ctx)
}

func (e *kube) DestroyStep(ctx context.Context, step *types.Step, taskUUID string) error {
	log.Trace().Str("taskUUID", taskUUID).Msgf("Stopping step: %s", step.Name)
	err := stopPod(ctx, e, step, defaultDeleteOptions)
	return err
}

// DestroyWorkflow destroys the pipeline environment.
func (e *kube) DestroyWorkflow(ctx context.Context, conf *types.Config, taskUUID string) error {
	log.Trace().Str("taskUUID", taskUUID).Msg("deleting Kubernetes primitives")

	// Use noContext because the ctx sent to this function will be canceled/done in case of error or canceled by user.
	for _, stage := range conf.Stages {
		for _, step := range stage.Steps {
			err := stopPod(ctx, e, step, defaultDeleteOptions)
			if err != nil {
				return err
			}

			if step.Type == types.StepTypeService {
				err := stopService(ctx, e, step, defaultDeleteOptions)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, vol := range conf.Volumes {
		err := stopVolume(ctx, e, vol.Name, defaultDeleteOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

func isPodPending(pod *v1.Pod) bool {
	cs := getFirstContainerState(pod)
	return pod.Status.Phase == v1.PodPending || (cs != nil && cs.Waiting != nil && len(cs.Waiting.Reason) > 0)
}

func isPodSucceed(pod *v1.Pod) bool {
	cs := getFirstContainerState(pod)
	return pod.Status.Phase == v1.PodSucceeded || (cs != nil && cs.Terminated != nil && cs.Terminated.ExitCode == 0)
}

func isPodFailed(pod *v1.Pod) bool {
	cs := getFirstContainerState(pod)
	return pod.Status.Phase == v1.PodFailed || (cs != nil && cs.Terminated != nil && cs.Terminated.ExitCode != 0)
}

func newPodFailedError(pod *v1.Pod) error {
	cs := getFirstContainerState(pod)
	if cs != nil && cs.Terminated != nil {
		return fmt.Errorf("pod %s failed because of %s, %s: %w", pod.Name, pod.Status.Reason, pod.Status.Message, newContainerTerminatedError(cs.Terminated))
	} else {
		return fmt.Errorf("pod %s failed because of %s, %s", pod.Name, pod.Status.Reason, pod.Status.Message)
	}
}

func newContainerTerminatedError(terminated *v1.ContainerStateTerminated) error {
	return fmt.Errorf("container %s is terminated because of %s, %s", terminated.ContainerID, terminated.Reason, terminated.Message)
}

func newPodPendingError(pod *v1.Pod) error {
	cs := getFirstContainerState(pod)
	if cs != nil && cs.Waiting != nil {
		return fmt.Errorf("pod %s pending because of %s, %s: %w", pod.Name, pod.Status.Reason, pod.Status.Message, newContainerWaitingError(cs.Waiting))
	} else {
		return fmt.Errorf("pod %s pending because of %s, %s", pod.Name, pod.Status.Reason, pod.Status.Message)
	}
}

func newContainerWaitingError(waiting *v1.ContainerStateWaiting) error {
	return fmt.Errorf("container is waiting because of %s, %s", waiting.Reason, waiting.Message)
}

func getFirstContainerState(pod *v1.Pod) *v1.ContainerState {
	if pod == nil {
		return nil
	}
	ps := pod.Status
	if len(ps.ContainerStatuses) == 0 {
		return nil
	}
	cs := ps.ContainerStatuses[0]
	return &cs.State
}
