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

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package config_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go.woodpecker-ci.org/woodpecker/v2/server/forge/mocks"
	forge_types "go.woodpecker-ci.org/woodpecker/v2/server/forge/types"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/server/services/config"
)

func TestFetch(t *testing.T) {
	t.Parallel()

	type file struct {
		name string
		data []byte
	}

	yamlPipeline := []byte(`
	steps:
		hello:
			image: alpine
			commands:
			- echo Hello alpine!
	`)
	jsonnetPipeline := []byte(`
		{
			steps: {
				hello: {
					image: "alpine",
					commands: [
						std.join(" ", ["echo", "Hello", self.image, "!"]),
					]
				},
			},
		}
	)`)

	testTable := []struct {
		name              string
		repoConfig        string
		files             []file
		expectedFileNames []string
		expectedError     bool
	}{
		{
			name:       "Default config - .woodpecker/",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker/text.txt",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/release.yml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/image.png",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker/release.yml",
			},
			expectedError: false,
		},
		{
			name:       "Default config with .yaml - .woodpecker/",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker/text.txt",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/release.yaml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/image.png",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker/release.yaml",
			},
			expectedError: false,
		},
		{
			name:       "Default config with .yaml, .yml mix - .woodpecker/",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker/text.txt",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/release.yaml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/other.yml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/image.png",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/notification.jsonnet",
				data: jsonnetPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker/release.yaml",
				".woodpecker/other.yml",
				".woodpecker/notification.jsonnet",
			},
			expectedError: false,
		},
		{
			name:       "Default config check .woodpecker.yaml before .woodpecker.yml",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker.yaml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker.yaml",
			},
			expectedError: false,
		},
		{
			name:       "Override via API with custom config",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker.yml",
			},
			expectedError: false,
		},
		{
			name:       "Use old config on 204 response",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker.yaml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker.yaml",
			},
			expectedError: false,
		},
		{
			name:              "Default config - Empty repo",
			repoConfig:        "",
			files:             []file{},
			expectedFileNames: []string{},
			expectedError:     true,
		},
		{
			name:       "Default config - Additional sub-folders",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker/test.yml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/sub-folder/config.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker/test.yml",
			},
			expectedError: false,
		},
		{
			name:       "Default config - Additional none .yml files",
			repoConfig: "",
			files: []file{{
				name: ".woodpecker/notes.txt",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/image.png",
				data: yamlPipeline,
			}, {
				name: ".woodpecker/test.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker/test.yml",
			},
			expectedError: false,
		},
		{
			name:       "Default config - Empty Folder",
			repoConfig: " ",
			files: []file{{
				name: ".woodpecker/.keep",
				data: yamlPipeline,
			}, {
				name: ".woodpecker.yml",
				data: nil,
			}, {
				name: ".woodpecker.yaml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".woodpecker.yaml",
			},
			expectedError: false,
		},
		{
			name:       "Special config - folder (ignoring default files)",
			repoConfig: ".my-ci-folder/",
			files: []file{{
				name: ".woodpecker/test.yml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker.yml",
				data: yamlPipeline,
			}, {
				name: ".woodpecker.yaml",
				data: yamlPipeline,
			}, {
				name: ".my-ci-folder/test.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".my-ci-folder/test.yml",
			},
			expectedError: false,
		},
		{
			name:       "Special config - folder",
			repoConfig: ".my-ci-folder/",
			files: []file{{
				name: ".my-ci-folder/test.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".my-ci-folder/test.yml",
			},
			expectedError: false,
		},
		{
			name:       "Special config - subfolder",
			repoConfig: ".my-ci-folder/my-config/",
			files: []file{{
				name: ".my-ci-folder/my-config/test.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".my-ci-folder/my-config/test.yml",
			},
			expectedError: false,
		},
		{
			name:       "Special config - file",
			repoConfig: ".config.yml",
			files: []file{{
				name: ".config.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".config.yml",
			},
			expectedError: false,
		},
		{
			name:       "Special config - file inside subfolder",
			repoConfig: ".my-ci-folder/sub-folder/config.yml",
			files: []file{{
				name: ".my-ci-folder/sub-folder/config.yml",
				data: yamlPipeline,
			}},
			expectedFileNames: []string{
				".my-ci-folder/sub-folder/config.yml",
			},
			expectedError: false,
		},
		{
			name:              "Special config - empty repo",
			repoConfig:        ".config.yml",
			files:             []file{},
			expectedFileNames: []string{},
			expectedError:     true,
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			repo := &model.Repo{Owner: "laszlocph", Name: "multipipeline", Config: tt.repoConfig}

			f := new(mocks.Forge)
			f.On("Name").Return("mockForge")

			dirs := map[string][]*forge_types.FileMeta{}
			for _, file := range tt.files {
				f.On("File", mock.Anything, mock.Anything, mock.Anything, mock.Anything, file.name).Once().Return(file.data, nil)
				path := filepath.Dir(file.name)
				if path != "." {
					dirs[path] = append(dirs[path], &forge_types.FileMeta{
						Name: file.name,
						Data: file.data,
					})
				}
			}

			for path, files := range dirs {
				f.On("Dir", mock.Anything, mock.Anything, mock.Anything, mock.Anything, path).Once().Return(files, nil)
			}

			// if the previous mocks do not match return not found errors
			f.On("File", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("file not found"))
			f.On("Dir", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("directory not found"))

			configFetcher := config.NewForge(
				time.Second*3,
				3,
			)
			files, err := configFetcher.Fetch(
				context.Background(),
				f,
				&model.User{Token: "xxx"},
				repo,
				&model.Pipeline{Commit: "89ab7b2d6bfb347144ac7c557e638ab402848fee"},
				nil,
				false,
			)
			if tt.expectedError && err == nil {
				t.Fatal("expected an error")
			} else if !tt.expectedError && err != nil {
				t.Fatal("error fetching config:", err)
			}

			matchingFiles := make([]string, len(files))
			for i := range files {
				matchingFiles[i] = files[i].Name
			}
			assert.ElementsMatch(t, tt.expectedFileNames, matchingFiles, "expected some other pipeline files")
		})
	}
}
