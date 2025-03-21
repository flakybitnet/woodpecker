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
	Copyright (c) 2023 Woodpecker Authors

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package lint

import (
	"context"
	"fmt"
	"go.woodpecker-ci.org/woodpecker/v2/shared/constant"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	"go.woodpecker-ci.org/woodpecker/v2/cli/common"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/yaml"
	"go.woodpecker-ci.org/woodpecker/v2/pipeline/frontend/yaml/linter"
)

// Command exports the info command.
var Command = &cli.Command{
	Name:      "lint",
	Usage:     "lint a pipeline configuration file",
	ArgsUsage: "[path/to/.woodpecker.yaml]",
	Action:    lint,
}

func lint(ctx context.Context, c *cli.Command) error {
	return common.RunPipelineFunc(ctx, c, lintFile, lintDir)
}

func lintDir(ctx context.Context, c *cli.Command, dir string) error {
	var errorStrings []string
	if err := filepath.Walk(dir, func(walkPath string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		// check if it is a regular file (not dir)
		if info.Mode().IsRegular() && slices.Contains(constant.SupportedConfigExtensions, path.Ext(info.Name())) {
			fmt.Println("#", info.Name())
			if err := lintFile(ctx, c, walkPath); err != nil {
				errorStrings = append(errorStrings, err.Error())
			}
			fmt.Println("")
			return nil
		}

		return nil
	}); err != nil {
		return err
	}

	if len(errorStrings) != 0 {
		return fmt.Errorf("ERRORS: %s", strings.Join(errorStrings, "; "))
	}
	return nil
}

func lintFile(_ context.Context, _ *cli.Command, file string) error {
	fi, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fi.Close()

	buf, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	rawConfig := string(buf)

	c, err := yaml.ParseString(rawConfig)
	if err != nil {
		return err
	}

	config := &linter.WorkflowConfig{
		File:      path.Base(file),
		RawConfig: rawConfig,
		Workflow:  c,
	}

	// TODO: lint multiple files at once to allow checks for sth like "depends_on" to work
	err = linter.New(linter.WithTrusted(true)).Lint([]*linter.WorkflowConfig{config})
	if err != nil {
		str, err := FormatLintError(config.File, err)

		if str != "" {
			fmt.Print(str)
		}

		return err
	}

	fmt.Println("✅ Config is valid")
	return nil
}
