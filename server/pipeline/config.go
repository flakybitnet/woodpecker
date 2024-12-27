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

package pipeline

import (
	forge_types "go.woodpecker-ci.org/woodpecker/v2/server/forge/types"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/server/pipeline/stepbuilder"
	"go.woodpecker-ci.org/woodpecker/v2/server/store"
)

func findOrPersistPipelineConfig(store store.Store, currentPipeline *model.Pipeline, forgeConfig *forge_types.FileMeta) (*model.Config, error) {
	return store.ConfigPersist(&model.Config{
		RepoID: currentPipeline.RepoID,
		Name:   stepbuilder.SanitizePath(forgeConfig.Name),
		Data:   forgeConfig.Data,
	})
}
