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

package gitea

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"

	"go.woodpecker-ci.org/woodpecker/v2/server/forge/types"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
)

const (
	hookEvent       = "X-Gitea-Event"
	hookPush        = "push"
	hookPullRequest = "pull_request"
	hookRelease     = "release"

	actionOpen  = "opened"
	actionSync  = "synchronized"
	actionClose = "closed"
)

// parseHook parses a Gitea hook from an http.Request and returns
// Repo and Pipeline detail. If a hook type is unsupported nil values are returned.
func parseHook(r *http.Request) (*model.Repo, *model.Pipeline, error) {
	hookType := r.Header.Get(hookEvent)
	switch hookType {
	case hookPush:
		return parsePushHook(r.Body)
	case hookPullRequest:
		return parsePullRequestHook(r.Body)
	case hookRelease:
		return parseReleaseHook(r.Body)
	}
	log.Debug().Msgf("unsupported hook type: '%s'", hookType)
	return nil, nil, &types.ErrIgnoreEvent{Event: hookType}
}

// parsePushHook parses a push hook and returns the Repo and Pipeline details.
// If the commit type is unsupported nil values are returned.
func parsePushHook(payload io.Reader) (repo *model.Repo, pipeline *model.Pipeline, err error) {
	push, err := parsePush(payload)
	if err != nil {
		return nil, nil, err
	}

	repo = toRepo(push.Repo)
	switch {
	case strings.HasPrefix(push.Ref, "refs/heads/"):
		pipeline = pipelineFromPush(push)
	case strings.HasPrefix(push.Ref, "refs/tags/"):
		// Gitea sends push hooks when tags are created.
		// The push hook contains more information than the tag created hook, so we choose to use the push hook for tags.
		if len(push.HeadCommit.ID) > 0 {
			pipeline = pipelineFromPushTag(push)
		} else {
			log.Debug().Str("ref", push.Ref).Msg("skipping unsupported deleted tag push event")
		}
	default:
		log.Debug().Str("ref", push.Ref).Msg("skipping unsupported hook push reference")
	}

	return repo, pipeline, err
}

// parsePullRequestHook parses a pull_request hook and returns the Repo and Pipeline details.
func parsePullRequestHook(payload io.Reader) (*model.Repo, *model.Pipeline, error) {
	var (
		repo     *model.Repo
		pipeline *model.Pipeline
	)

	pr, err := parsePullRequest(payload)
	if err != nil {
		return nil, nil, err
	}

	if pr.PullRequest == nil {
		// this should never have happened, but it did - so we check
		return nil, nil, fmt.Errorf("parsed pull_request webhook does not contain pull_request info")
	}

	// Don't trigger pipelines for non-code changes ...
	if pr.Action != actionOpen && pr.Action != actionSync && pr.Action != actionClose {
		log.Debug().Msgf("pull_request action is '%s' and no open or sync", pr.Action)
		return nil, nil, nil
	}

	repo = toRepo(pr.Repo)
	pipeline = pipelineFromPullRequest(pr)
	return repo, pipeline, err
}

// parseReleaseHook parses a release hook and returns the Repo and Pipeline details.
func parseReleaseHook(payload io.Reader) (*model.Repo, *model.Pipeline, error) {
	var (
		repo     *model.Repo
		pipeline *model.Pipeline
	)

	release, err := parseRelease(payload)
	if err != nil {
		return nil, nil, err
	}

	repo = toRepo(release.Repo)
	pipeline = pipelineFromRelease(release)
	return repo, pipeline, err
}
