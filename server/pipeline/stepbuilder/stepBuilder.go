/*
This file is part of Woodpecker CI.
Copyright (c) 2024 Woodpecker Authors

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, version 3 of the License.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

This file incorporates work covered by the following copyright and permission notice:
	Copyright (c) 2022 Woodpecker Authors
	Copyright (c) 2018 Drone.IO Inc.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package stepbuilder

import (
	"fmt"
	"go.woodpecker-ci.org/woodpecker/v2/shared/constant"
	"path/filepath"
	"strings"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"

	backend_types "go.woodpecker-ci.org/woodpecker/v2/pipeline/backend/types"
	pipeline_errors "go.woodpecker-ci.org/woodpecker/v2/pipeline/errors"
	errorTypes "go.woodpecker-ci.org/woodpecker/v2/pipeline/errors/types"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/metadata"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/yaml"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/yaml/compiler"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/yaml/linter"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/yaml/matrix"
	yaml_types "go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/yaml/types"
	"go.woodpecker-ci.org/woodpecker/v2/server"
	forge_types "go.woodpecker-ci.org/woodpecker/v2/server/forge/types"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
)

// StepBuilder Takes the hook data and the yaml and returns in internal data model.
type StepBuilder struct {
	Repo      *model.Repo
	Curr      *model.Pipeline
	Last      *model.Pipeline
	Netrc     *model.Netrc
	Secs      []*model.Secret
	Regs      []*model.Registry
	Host      string
	Configs   []*forge_types.FileMeta // YAML or JSON
	Envs      map[string]string
	Forge     metadata.ServerForge
	ProxyOpts compiler.ProxyOptions
}

type Item struct {
	Workflow  *model.Workflow
	Labels    map[string]string
	DependsOn []string
	RunsOn    []string
	Config    *backend_types.Config
}

func (b *StepBuilder) Build() (items []*Item, errorsAndWarnings error) {
	b.Configs = forge_types.SortByName(b.Configs)

	pidSequence := 1

	for _, config := range b.Configs {
		// matrix axes
		axes, err := matrix.ParseString(string(config.Data))
		if err != nil {
			return nil, err
		}
		if len(axes) == 0 {
			axes = append(axes, matrix.Axis{})
		}

		for i, axis := range axes {
			workflow := &model.Workflow{
				PID:     pidSequence,
				State:   model.StatusPending,
				Environ: axis,
				Name:    SanitizePath(config.Name),
			}
			if len(axes) > 1 {
				workflow.AxisID = i + 1
			}
			item, err := b.genItemForWorkflow(workflow, axis, string(config.Data))
			if err != nil && pipeline_errors.HasBlockingErrors(err) {
				return nil, err
			} else if err != nil {
				errorsAndWarnings = multierr.Append(errorsAndWarnings, err)
			}

			if item == nil {
				continue
			}
			items = append(items, item)
			pidSequence++
		}

		// TODO: add summary workflow that send status back based on workflows generated by matrix function
		// depend on https://github.com/woodpecker-ci/woodpecker/issues/778
	}

	items = filterItemsWithMissingDependencies(items)

	// check if at least one step can start if slice is not empty
	if len(items) > 0 && !stepListContainsItemsToRun(items) {
		return nil, fmt.Errorf("pipeline has no steps to run")
	}

	return items, errorsAndWarnings
}

func (b *StepBuilder) genItemForWorkflow(workflow *model.Workflow, axis matrix.Axis, data string) (item *Item, errorsAndWarnings error) {
	workflowMetadata := MetadataFromStruct(b.Forge, b.Repo, b.Curr, b.Last, workflow, b.Host)
	environ := b.environmentVariables(workflowMetadata, axis)

	// add global environment variables for substituting
	for k, v := range b.Envs {
		if _, exists := environ[k]; exists {
			// don't override existing values
			continue
		}
		environ[k] = v
	}

	// substitute vars
	substituted, err := metadata.EnvVarSubst(data, environ)
	if err != nil {
		return nil, multierr.Append(errorsAndWarnings, err)
	}

	// parse yaml pipeline
	parsed, err := yaml.ParseString(substituted)
	if err != nil {
		return nil, &errorTypes.PipelineError{Message: err.Error(), Type: errorTypes.PipelineErrorTypeCompiler}
	}

	// lint pipeline
	errorsAndWarnings = multierr.Append(errorsAndWarnings, linter.New(
		linter.WithTrusted(b.Repo.IsTrusted),
		linter.PrivilegedPlugins(server.Config.Pipeline.Privileged),
	).Lint([]*linter.WorkflowConfig{{
		Workflow:  parsed,
		File:      workflow.Name,
		RawConfig: data,
	}}))
	if pipeline_errors.HasBlockingErrors(errorsAndWarnings) {
		return nil, errorsAndWarnings
	}

	// checking if filtered.
	if match, err := parsed.When.Match(workflowMetadata, true, environ); !match && err == nil {
		log.Debug().Str("pipeline", workflow.Name).Msg(
			"marked as skipped, does not match metadata",
		)
		return nil, nil
	} else if err != nil {
		log.Debug().Str("pipeline", workflow.Name).Msg(
			"pipeline config could not be parsed",
		)
		return nil, multierr.Append(errorsAndWarnings, err)
	}

	ir, err := b.toInternalRepresentation(parsed, environ, workflowMetadata, workflow.ID)
	if err != nil {
		return nil, multierr.Append(errorsAndWarnings, err)
	}

	if len(ir.Stages) == 0 {
		return nil, nil
	}

	item = &Item{
		Workflow:  workflow,
		Config:    ir,
		Labels:    parsed.Labels,
		DependsOn: parsed.DependsOn,
		RunsOn:    parsed.RunsOn,
	}
	if item.Labels == nil {
		item.Labels = map[string]string{}
	}

	return item, errorsAndWarnings
}

func stepListContainsItemsToRun(items []*Item) bool {
	for i := range items {
		if items[i].Workflow.State == model.StatusPending {
			return true
		}
	}
	return false
}

func filterItemsWithMissingDependencies(items []*Item) []*Item {
	itemsToRemove := make([]*Item, 0)

	for _, item := range items {
		for _, dep := range item.DependsOn {
			if !containsItemWithName(dep, items) {
				itemsToRemove = append(itemsToRemove, item)
			}
		}
	}

	if len(itemsToRemove) > 0 {
		filtered := make([]*Item, 0)
		for _, item := range items {
			if !containsItemWithName(item.Workflow.Name, itemsToRemove) {
				filtered = append(filtered, item)
			}
		}
		// Recursive to handle transitive deps
		return filterItemsWithMissingDependencies(filtered)
	}

	return items
}

func containsItemWithName(name string, items []*Item) bool {
	for _, item := range items {
		if name == item.Workflow.Name {
			return true
		}
	}
	return false
}

func (b *StepBuilder) environmentVariables(metadata metadata.Metadata, axis matrix.Axis) map[string]string {
	environ := metadata.Environ()
	for k, v := range axis {
		environ[k] = v
	}
	return environ
}

func (b *StepBuilder) toInternalRepresentation(parsed *yaml_types.Workflow, environ map[string]string, metadata metadata.Metadata, workflowID int64) (*backend_types.Config, error) {
	var secrets []compiler.Secret
	for _, sec := range b.Secs {
		var events []string
		for _, event := range sec.Events {
			events = append(events, string(event))
		}

		secrets = append(secrets, compiler.Secret{
			Name:           sec.Name,
			Value:          sec.Value,
			AllowedPlugins: sec.Images,
			Events:         events,
		})
	}

	var registries []compiler.Registry
	for _, reg := range b.Regs {
		registries = append(registries, compiler.Registry{
			Hostname: reg.Address,
			Username: reg.Username,
			Password: reg.Password,
		})
	}

	return compiler.New(
		compiler.WithEnviron(environ),
		compiler.WithEnviron(b.Envs),
		// TODO: server deps should be moved into StepBuilder fields and set on StepBuilder creation
		compiler.WithEscalated(server.Config.Pipeline.Privileged...),
		compiler.WithResourceLimit(server.Config.Pipeline.Limits.MemSwapLimit, server.Config.Pipeline.Limits.MemLimit, server.Config.Pipeline.Limits.ShmSize, server.Config.Pipeline.Limits.CPUQuota, server.Config.Pipeline.Limits.CPUShares, server.Config.Pipeline.Limits.CPUSet),
		compiler.WithVolumes(server.Config.Pipeline.Volumes...),
		compiler.WithNetworks(server.Config.Pipeline.Networks...),
		compiler.WithLocal(false),
		compiler.WithOption(
			compiler.WithNetrc(
				b.Netrc.Login,
				b.Netrc.Password,
				b.Netrc.Machine,
			),
			b.Repo.IsSCMPrivate || server.Config.Pipeline.AuthenticatePublicRepos,
		),
		compiler.WithDefaultCloneImage(server.Config.Pipeline.DefaultCloneImage),
		compiler.WithRegistry(registries...),
		compiler.WithSecret(secrets...),
		compiler.WithPrefix(
			fmt.Sprintf(
				"wp_%s_%d",
				strings.ToLower(ulid.Make().String()),
				workflowID,
			),
		),
		compiler.WithProxy(b.ProxyOpts),
		compiler.WithWorkspaceFromURL(compiler.DefaultWorkspaceBase, b.Repo.ForgeURL),
		compiler.WithMetadata(metadata),
		compiler.WithTrusted(b.Repo.IsTrusted),
		compiler.WithNetrcOnlyTrusted(b.Repo.NetrcOnlyTrusted),
	).Compile(parsed)
}

func SanitizePath(path string) string {
	path = filepath.Base(path)
	path = strings.TrimPrefix(path, ".")
	for _, ext := range constant.SupportedConfigExtensions {
		path = strings.TrimSuffix(path, ext)
	}
	return path
}
