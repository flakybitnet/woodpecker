/*
This file is part of Woodpecker CI.
Copyright (c) 2025 Woodpecker Authors

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
	Copyright (c) 2024 Woodpecker Authors

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/sethvargo/go-retry"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.woodpecker-ci.org/woodpecker/v2/server/forge"
	"go.woodpecker-ci.org/woodpecker/v2/server/forge/types"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/shared/constant"
)

type forgeFetcher struct {
	timeout    time.Duration
	retryCount uint
}

func NewForge(timeout time.Duration, retries uint) Service {
	return &forgeFetcher{
		timeout:    timeout,
		retryCount: retries,
	}
}

func (f *forgeFetcher) Fetch(ctx context.Context, forge forge.Forge, user *model.User, repo *model.Repo, pipeline *model.Pipeline, oldConfigData []*types.FileMeta, restart bool) (files []*types.FileMeta, err error) {
	// skip fetching if we are restarting and have the old config
	if restart && len(oldConfigData) > 0 {
		return oldConfigData, nil
	}

	ffc := &forgeFetcherContext{
		forge:    forge,
		user:     user,
		repo:     repo,
		pipeline: pipeline,
		timeout:  f.timeout,
	}

	backoff := retry.WithMaxRetries(uint64(f.retryCount), retry.NewConstant(100*time.Millisecond))
	err = retry.Do(ctx, backoff, func(ctx context.Context) error {
		files, err = ffc.fetch(ctx, strings.TrimSpace(repo.Config))
		if err != nil {
			return retry.RetryableError(err)
		}
		return nil
	})

	return
}

type forgeFetcherContext struct {
	forge    forge.Forge
	user     *model.User
	repo     *model.Repo
	pipeline *model.Pipeline
	timeout  time.Duration
}

// fetch attempts to fetch the configuration file(s) for the given config string.
func (f *forgeFetcherContext) fetch(c context.Context, config string) ([]*types.FileMeta, error) {
	ctx, cancel := context.WithTimeout(c, f.timeout)
	defer cancel()

	var err error
	var configFiles []*types.FileMeta
	if len(config) > 0 {
		log.Trace().Str("repository", f.repo.FullName).Str("config", config).Msg("use user config")

		// could be adapted to allow the user to supply a list like we do in the defaults
		configs := []string{config}

		configFiles, err = f.getFirstAvailableConfig(ctx, configs)
		if err != nil {
			return nil, fmt.Errorf("user defined config '%s' not found: %w", config, err)
		}

	} else {
		log.Trace().Str("repository", f.repo.FullName).Msg("user did not define own config, following default procedure")
		// for the order see shared/constants/constants.go
		configFiles, err = f.getFirstAvailableConfig(ctx, constant.DefaultConfigOrder[:])
		if err != nil {
			return nil, fmt.Errorf("fallback config not found: %w", err)
		}
	}

	return configFiles, nil
}

func (f *forgeFetcherContext) fetchConfigFile(ctx context.Context, config string) ([]*types.FileMeta, error) {
	log.Trace().Str("file", config).Msg("fetching from forge")

	file, err := f.forge.File(ctx, f.user, f.repo, f.pipeline, config)
	if err != nil {
		return nil, err
	}

	files := []*types.FileMeta{{
		Name: config,
		Data: file,
	}}

	return files, nil
}

func (f *forgeFetcherContext) fetchConfigDir(ctx context.Context, config string) ([]*types.FileMeta, error) {
	log.Trace().Str("dir", config).Msg("fetching from forge")

	files, err := f.forge.Dir(ctx, f.user, f.repo, f.pipeline, strings.TrimSuffix(config, "/"))
	if errors.Is(err, types.ErrNotImplemented) {
		log.Error().Err(err).Str("forge", f.forge.Name()).Str("repository", f.repo.FullName).
			Str("config-dir", config).Msg("dir fetching is not implemented")
	}

	return files, err
}

func (f *forgeFetcherContext) getFirstAvailableConfig(ctx context.Context, configs []string) (configFiles []*types.FileMeta, err error) {
	var forgeErr []error

	for _, config := range configs {
		var files []*types.FileMeta

		if strings.HasSuffix(config, "/") { // config is a folder
			files, err = f.fetchConfigDir(ctx, config)
			if err != nil {
				// if folder is not supported we will get a "Not implemented" error and continue
				if !(errors.Is(err, types.ErrNotImplemented) || errors.Is(err, &types.ErrConfigNotFound{})) {
					forgeErr = append(forgeErr, err)
				}
				continue
			}

		} else { // config is a file
			files, err = f.fetchConfigFile(ctx, config)
			if err != nil {
				if !errors.Is(err, &types.ErrConfigNotFound{}) {
					forgeErr = append(forgeErr, err)
				}
				continue
			}
		}

		supportedConfigs := f.filterSupportedConfigs(files)
		if len(supportedConfigs) > 0 {
			return supportedConfigs, nil
		}
	}

	// got unexpected errors
	if len(forgeErr) != 0 {
		return nil, errors.Join(forgeErr...)
	}

	// nothing found
	return nil, &types.ErrConfigNotFound{Configs: configs}
}

func (f *forgeFetcherContext) filterSupportedConfigs(files []*types.FileMeta) (configFiles []*types.FileMeta) {
	for _, file := range files {
		if !slices.Contains(constant.SupportedConfigExtensions, path.Ext(file.Name)) {
			log.Trace().Str("file", file.Name).Msg("skipped unsupported config format")
			continue
		}
		if len(file.Data) == 0 {
			log.Trace().Str("file", file.Name).Msg("skipped empty config file")
			continue
		}
		configFiles = append(configFiles, file)
	}
	return
}
