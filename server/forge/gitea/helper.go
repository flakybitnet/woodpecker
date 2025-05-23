// Copyright 2022 Woodpecker Authors
// Copyright 2018 Drone.IO Inc.
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

package gitea

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"

	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/shared/utils"
)

// toRepo converts a Gitea repository to a Woodpecker repository.
func toRepo(from *gitea.Repository) *model.Repo {
	name := strings.Split(from.FullName, "/")[1]
	avatar := expandAvatar(
		from.HTMLURL,
		from.Owner.AvatarURL,
	)
	return &model.Repo{
		ForgeRemoteID: model.ForgeRemoteID(fmt.Sprint(from.ID)),
		SCMKind:       model.RepoGit,
		Name:          name,
		Owner:         from.Owner.UserName,
		FullName:      from.FullName,
		Avatar:        avatar,
		ForgeURL:      from.HTMLURL,
		IsSCMPrivate:  from.Private || from.Owner.Visibility != gitea.VisibleTypePublic,
		Clone:         from.CloneURL,
		CloneSSH:      from.SSHURL,
		Branch:        from.DefaultBranch,
		Perm:          toPerm(from.Permissions),
		PREnabled:     from.HasPullRequests,
	}
}

// toPerm converts a Gitea permission to a Woodpecker permission.
func toPerm(from *gitea.Permission) *model.Perm {
	return &model.Perm{
		Pull:  from.Pull,
		Push:  from.Push,
		Admin: from.Admin,
	}
}

// toTeam converts a Gitea team to a Woodpecker team.
func toTeam(from *gitea.Organization, link string) *model.Team {
	return &model.Team{
		Login:  from.UserName,
		Avatar: expandAvatar(link, from.AvatarURL),
	}
}

// pipelineFromPush extracts the Pipeline data from a Gitea push hook.
func pipelineFromPush(hook *pushHook) *model.Pipeline {
	link := hook.HeadCommit.URL
	if len(hook.Commits) > 1 {
		link = hook.Compare
	}
	message, _, _ := strings.Cut(hook.HeadCommit.Message, "\n")

	return &model.Pipeline{
		Event:        model.EventPush,
		Commit:       hook.After,
		Ref:          hook.Ref,
		ForgeURL:     link,
		Branch:       strings.TrimPrefix(hook.Ref, "refs/heads/"),
		Message:      message,
		Avatar:       expandAvatar(hook.Repo.HTMLURL, fixMalformedAvatar(hook.Sender.AvatarURL)),
		Author:       hook.Sender.UserName,
		Sender:       hook.Sender.UserName,
		Email:        hook.Sender.Email,
		Timestamp:    time.Now().UTC().Unix(),
		ChangedFiles: getChangedFilesFromPushHook(hook),
	}
}

func getChangedFilesFromPushHook(hook *pushHook) []string {
	// assume a capacity of 4 changed files per commit
	files := make([]string, 0, len(hook.Commits)*4)
	for _, c := range hook.Commits {
		files = append(files, c.Added...)
		files = append(files, c.Removed...)
		files = append(files, c.Modified...)
	}

	files = append(files, hook.HeadCommit.Added...)
	files = append(files, hook.HeadCommit.Removed...)
	files = append(files, hook.HeadCommit.Modified...)

	return utils.DeduplicateStrings(files)
}

// pipelineFromPushTag extracts the Pipeline data from a Gitea push tag hook.
func pipelineFromPushTag(hook *pushHook) *model.Pipeline {
	tag := strings.TrimPrefix(hook.Ref, "refs/tags/")
	message, _, _ := strings.Cut(hook.HeadCommit.Message, "\n")
	return &model.Pipeline{
		Event:        model.EventTag,
		Commit:       hook.After,
		Ref:          hook.Ref,
		ForgeURL:     fmt.Sprintf("%s/src/tag/%s", hook.Repo.HTMLURL, tag),
		Message:      message,
		Avatar:       expandAvatar(hook.Repo.HTMLURL, fixMalformedAvatar(hook.Sender.AvatarURL)),
		Author:       hook.Sender.UserName,
		Sender:       hook.Sender.UserName,
		Email:        hook.Sender.Email,
		Timestamp:    time.Now().UTC().Unix(),
		ChangedFiles: getChangedFilesFromPushHook(hook),
	}
}

// pipelineFromPullRequest extracts the Pipeline data from a Gitea pull_request hook.
func pipelineFromPullRequest(hook *pullRequestHook) *model.Pipeline {
	avatar := expandAvatar(
		hook.Repo.HTMLURL,
		fixMalformedAvatar(hook.PullRequest.Poster.AvatarURL),
	)

	event := model.EventPull
	if hook.Action == actionClose {
		event = model.EventPullClosed
	}

	pipeline := &model.Pipeline{
		Event:    event,
		Commit:   hook.PullRequest.Head.Sha,
		ForgeURL: hook.PullRequest.HTMLURL,
		Ref:      fmt.Sprintf("refs/pull/%d/head", hook.Number),
		Branch:   hook.PullRequest.Base.Ref,
		Message:  hook.PullRequest.Title,
		Author:   hook.PullRequest.Poster.UserName,
		Avatar:   avatar,
		Sender:   hook.Sender.UserName,
		Email:    hook.Sender.Email,
		Title:    hook.PullRequest.Title,
		Refspec: fmt.Sprintf("%s:%s",
			hook.PullRequest.Head.Ref,
			hook.PullRequest.Base.Ref,
		),
		PullRequestLabels: convertLabels(hook.PullRequest.Labels),
		FromFork:          hook.PullRequest.Head.RepoID != hook.PullRequest.Base.RepoID,
	}

	return pipeline
}

func pipelineFromRelease(hook *releaseHook) *model.Pipeline {
	avatar := expandAvatar(
		hook.Repo.HTMLURL,
		fixMalformedAvatar(hook.Sender.AvatarURL),
	)

	return &model.Pipeline{
		Event:        model.EventRelease,
		Ref:          fmt.Sprintf("refs/tags/%s", hook.Release.TagName),
		ForgeURL:     hook.Release.HTMLURL,
		Branch:       hook.Release.Target,
		Message:      fmt.Sprintf("created release %s", hook.Release.Title),
		Avatar:       avatar,
		Author:       hook.Sender.UserName,
		Sender:       hook.Sender.UserName,
		Email:        hook.Sender.Email,
		IsPrerelease: hook.Release.IsPrerelease,
	}
}

// parsePush parses a push hook from a read closer.
func parsePush(r io.Reader) (*pushHook, error) {
	push := new(pushHook)
	err := json.NewDecoder(r).Decode(push)
	return push, err
}

func parsePullRequest(r io.Reader) (*pullRequestHook, error) {
	pr := new(pullRequestHook)
	err := json.NewDecoder(r).Decode(pr)
	return pr, err
}

func parseRelease(r io.Reader) (*releaseHook, error) {
	pr := new(releaseHook)
	err := json.NewDecoder(r).Decode(pr)
	return pr, err
}

// fixMalformedAvatar fixes an avatar url if malformed (currently a known bug with gitea).
func fixMalformedAvatar(url string) string {
	index := strings.Index(url, "///")
	if index != -1 {
		return url[index+1:]
	}
	index = strings.Index(url, "//avatars/")
	if index != -1 {
		return strings.ReplaceAll(url, "//avatars/", "/avatars/")
	}
	return url
}

// expandAvatar converts a relative avatar URL to the absolute url.
func expandAvatar(repo, rawURL string) string {
	aURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if aURL.IsAbs() {
		// Url is already absolute
		return aURL.String()
	}

	// Resolve to base
	burl, err := url.Parse(repo)
	if err != nil {
		return rawURL
	}
	aURL = burl.ResolveReference(aURL)

	return aURL.String()
}

// matchingHooks return matching hooks.
func matchingHooks(hooks []*gitea.Hook, rawURL string) *gitea.Hook {
	link, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	for _, hook := range hooks {
		if val, ok := hook.Config["url"]; ok {
			hookURL, err := url.Parse(val)
			if err == nil && hookURL.Host == link.Host {
				return hook
			}
		}
	}
	return nil
}

func convertLabels(from []*gitea.Label) []string {
	labels := make([]string, len(from))
	for i, label := range from {
		labels[i] = label.Name
	}
	return labels
}
