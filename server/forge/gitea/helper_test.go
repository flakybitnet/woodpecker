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
	"bytes"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/franela/goblin"

	"go.woodpecker-ci.org/woodpecker/v2/server/forge/gitea/fixtures"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/shared/utils"
)

func Test_parse(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("Gitea", func() {
		g.It("Should parse push hook payload", func() {
			buf := bytes.NewBufferString(fixtures.HookPush)
			hook, err := parsePush(buf)
			g.Assert(err).IsNil()
			g.Assert(hook.Ref).Equal("refs/heads/main")
			g.Assert(hook.After).Equal("ef98532add3b2feb7a137426bba1248724367df5")
			g.Assert(hook.Before).Equal("4b2626259b5a97b6b4eab5e6cca66adb986b672b")
			g.Assert(hook.Compare).Equal("http://gitea.golang.org/gordon/hello-world/compare/4b2626259b5a97b6b4eab5e6cca66adb986b672b...ef98532add3b2feb7a137426bba1248724367df5")
			g.Assert(hook.Repo.Name).Equal("hello-world")
			g.Assert(hook.Repo.HTMLURL).Equal("http://gitea.golang.org/gordon/hello-world")
			g.Assert(hook.Repo.Owner.UserName).Equal("gordon")
			g.Assert(hook.Repo.FullName).Equal("gordon/hello-world")
			g.Assert(hook.Repo.Owner.Email).Equal("gordon@golang.org")
			g.Assert(hook.Repo.Private).Equal(true)
			g.Assert(hook.Pusher.Email).Equal("gordon@golang.org")
			g.Assert(hook.Pusher.UserName).Equal("gordon")
			g.Assert(hook.Sender.UserName).Equal("gordon")
			g.Assert(hook.Sender.AvatarURL).Equal("http://gitea.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
		})

		g.It("Should parse tag push hook payload", func() {
			buf := bytes.NewBufferString(fixtures.HookPushTag)
			hook, err := parsePush(buf)
			g.Assert(err).IsNil()
			g.Assert(hook.Ref).Equal("refs/tags/v1.0.0")
			g.Assert(hook.Sha).Equal("")
			g.Assert(hook.After).Equal("28c3613ae62640216bea5e7dc71aa65356e4298b")
			g.Assert(hook.Repo.Name).Equal("woodpecktester")
			g.Assert(hook.Repo.HTMLURL).Equal("https://codeberg.org/meisam/woodpecktester")
			g.Assert(hook.Repo.FullName).Equal("meisam/woodpecktester")
			g.Assert(hook.Repo.Owner.Email).Equal("meisam@noreply.codeberg.org")
			g.Assert(hook.Repo.Owner.UserName).Equal("meisam")
			g.Assert(hook.Repo.Private).Equal(false)
			g.Assert(hook.Sender.UserName).Equal("6543")
			g.Assert(hook.Sender.AvatarURL).Equal("https://codeberg.org/avatars/09a234c768cb9bca78f6b2f82d6af173")
		})

		g.It("Should parse pull_request hook payload", func() {
			buf := bytes.NewBufferString(fixtures.HookPullRequest)
			hook, err := parsePullRequest(buf)
			g.Assert(err).IsNil()
			g.Assert(hook.Action).Equal("opened")
			g.Assert(hook.Number).Equal(int64(1))

			g.Assert(hook.Repo.Name).Equal("hello-world")
			g.Assert(hook.Repo.HTMLURL).Equal("http://gitea.golang.org/gordon/hello-world")
			g.Assert(hook.Repo.FullName).Equal("gordon/hello-world")
			g.Assert(hook.Repo.Owner.Email).Equal("gordon@golang.org")
			g.Assert(hook.Repo.Owner.UserName).Equal("gordon")
			g.Assert(hook.Repo.Private).Equal(true)
			g.Assert(hook.Sender.UserName).Equal("gordon")
			g.Assert(hook.Sender.AvatarURL).Equal("https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")

			g.Assert(hook.PullRequest.Title).Equal("Update the README with new information")
			g.Assert(hook.PullRequest.Body).Equal("please merge")
			g.Assert(hook.PullRequest.State).Equal(gitea.StateOpen)
			g.Assert(hook.PullRequest.Poster.UserName).Equal("gordon")
			g.Assert(hook.PullRequest.Base.Name).Equal("main")
			g.Assert(hook.PullRequest.Base.Ref).Equal("main")
			g.Assert(hook.PullRequest.Head.Name).Equal("feature/changes")
			g.Assert(hook.PullRequest.Head.Ref).Equal("feature/changes")
		})

		g.It("Should return a Pipeline struct from a push hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPush)
			hook, _ := parsePush(buf)
			pipeline := pipelineFromPush(hook)
			g.Assert(pipeline.Event).Equal(model.EventPush)
			g.Assert(pipeline.Commit).Equal(hook.After)
			g.Assert(pipeline.Ref).Equal(hook.Ref)
			g.Assert(pipeline.ForgeURL).Equal(hook.HeadCommit.URL)
			g.Assert(pipeline.Branch).Equal("main")
			g.Assert(pipeline.Message).Equal("bump")
			g.Assert(pipeline.Avatar).Equal("http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
			g.Assert(pipeline.Author).Equal(hook.Sender.UserName)
			g.Assert(utils.EqualSliceValues(pipeline.ChangedFiles, []string{"CHANGELOG.md", "app/controller/application.rb"})).IsTrue()
		})

		g.It("Should return a Repo struct from a push hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPush)
			hook, _ := parsePush(buf)
			repo := toRepo(hook.Repo)
			g.Assert(repo.Name).Equal(hook.Repo.Name)
			g.Assert(repo.Owner).Equal(hook.Repo.Owner.UserName)
			g.Assert(repo.FullName).Equal("gordon/hello-world")
			g.Assert(repo.ForgeURL).Equal(hook.Repo.HTMLURL)
		})

		g.It("Should return a Pipeline struct from a tag push hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPushTag)
			hook, _ := parsePush(buf)
			pipeline := pipelineFromPushTag(hook)
			g.Assert(pipeline.Event).Equal(model.EventTag)
			g.Assert(pipeline.Commit).Equal(hook.After)
			g.Assert(pipeline.Ref).Equal(hook.Ref)
			g.Assert(pipeline.ForgeURL).Equal("https://codeberg.org/meisam/woodpecktester/src/tag/v1.0.0")
			g.Assert(pipeline.Branch).Equal("")
			g.Assert(pipeline.Message).Equal("Fixed '.woodpecker/.check.yml'")
			g.Assert(pipeline.Avatar).Equal("https://codeberg.org/avatars/09a234c768cb9bca78f6b2f82d6af173")
			g.Assert(pipeline.Author).Equal(hook.Sender.UserName)
			g.Assert(utils.EqualSliceValues(pipeline.ChangedFiles, []string{".woodpecker/.check.yml"})).IsTrue()

			//g.Assert(pipeline.Commit).Equal(hook.Sha)
			//g.Assert(pipeline.Ref).Equal("refs/tags/v1.0.0")
			//g.Assert(pipeline.Branch).Equal("")
			//g.Assert(pipeline.ForgeURL).Equal("http://gitea.golang.org/gordon/hello-world/src/tag/v1.0.0")
			//g.Assert(pipeline.Message).Equal("Tag: v1.0.0")
		})

		g.It("Should return a Pipeline struct from a pull_request hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPullRequest)
			hook, _ := parsePullRequest(buf)
			pipeline := pipelineFromPullRequest(hook)
			g.Assert(pipeline.Event).Equal(model.EventPull)
			g.Assert(pipeline.Commit).Equal(hook.PullRequest.Head.Sha)
			g.Assert(pipeline.Ref).Equal("refs/pull/1/head")
			g.Assert(pipeline.ForgeURL).Equal("http://gitea.golang.org/gordon/hello-world/pull/1")
			g.Assert(pipeline.Branch).Equal("main")
			g.Assert(pipeline.Refspec).Equal("feature/changes:main")
			g.Assert(pipeline.Message).Equal(hook.PullRequest.Title)
			g.Assert(pipeline.Avatar).Equal("http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87")
			g.Assert(pipeline.Author).Equal(hook.PullRequest.Poster.UserName)
		})

		g.It("Should return a Repo struct from a pull_request hook", func() {
			buf := bytes.NewBufferString(fixtures.HookPullRequest)
			hook, _ := parsePullRequest(buf)
			repo := toRepo(hook.Repo)
			g.Assert(repo.Name).Equal(hook.Repo.Name)
			g.Assert(repo.Owner).Equal(hook.Repo.Owner.UserName)
			g.Assert(repo.FullName).Equal("gordon/hello-world")
			g.Assert(repo.ForgeURL).Equal(hook.Repo.HTMLURL)
		})

		g.It("Should return a Perm struct from a Gitea Perm", func() {
			perms := []gitea.Permission{
				{
					Admin: true,
					Push:  true,
					Pull:  true,
				},
				{
					Admin: true,
					Push:  true,
					Pull:  false,
				},
				{
					Admin: true,
					Push:  false,
					Pull:  false,
				},
			}
			for _, from := range perms {
				perm := toPerm(&from)
				g.Assert(perm.Pull).Equal(from.Pull)
				g.Assert(perm.Push).Equal(from.Push)
				g.Assert(perm.Admin).Equal(from.Admin)
			}
		})

		g.It("Should return a Team struct from a Gitea Org", func() {
			from := &gitea.Organization{
				UserName:  "woodpecker",
				AvatarURL: "/avatars/1",
			}

			to := toTeam(from, "http://localhost:80")
			g.Assert(to.Login).Equal(from.UserName)
			g.Assert(to.Avatar).Equal("http://localhost:80/avatars/1")
		})

		g.It("Should return a Repo struct from a Gitea Repo", func() {
			from := gitea.Repository{
				FullName: "gophers/hello-world",
				Owner: &gitea.User{
					UserName:  "gordon",
					AvatarURL: "http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				CloneURL:      "http://gitea.golang.org/gophers/hello-world.git",
				HTMLURL:       "http://gitea.golang.org/gophers/hello-world",
				Private:       true,
				DefaultBranch: "main",
				Permissions:   &gitea.Permission{Admin: true},
			}
			repo := toRepo(&from)
			g.Assert(repo.FullName).Equal(from.FullName)
			g.Assert(repo.Owner).Equal(from.Owner.UserName)
			g.Assert(repo.Name).Equal("hello-world")
			g.Assert(repo.Branch).Equal("main")
			g.Assert(repo.ForgeURL).Equal(from.HTMLURL)
			g.Assert(repo.Clone).Equal(from.CloneURL)
			g.Assert(repo.Avatar).Equal(from.Owner.AvatarURL)
			g.Assert(repo.IsSCMPrivate).Equal(from.Private)
			g.Assert(repo.Perm.Admin).IsTrue()
		})

		g.It("Should correct a malformed avatar url", func() {
			urls := []struct {
				Before string
				After  string
			}{
				{
					"http://gitea.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"http://gitea.golang.org/avatars/1",
					"http://gitea.golang.org/avatars/1",
				},
				{
					"http://gitea.golang.org//avatars/1",
					"http://gitea.golang.org/avatars/1",
				},
			}

			for _, url := range urls {
				got := fixMalformedAvatar(url.Before)
				g.Assert(got).Equal(url.After)
			}
		})

		g.It("Should expand the avatar url", func() {
			urls := []struct {
				Before string
				After  string
			}{
				{
					"/avatars/1",
					"http://gitea.io/avatars/1",
				},
				{
					"//1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
					"http://1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87",
				},
				{
					"/gitea/avatars/2",
					"http://gitea.io/gitea/avatars/2",
				},
			}

			repo := "http://gitea.io/foo/bar"
			for _, url := range urls {
				got := expandAvatar(repo, url.Before)
				g.Assert(got).Equal(url.After)
			}
		})
	})
}
