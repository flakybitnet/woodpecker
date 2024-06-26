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

package cron

import (
	"strconv"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/server/store"
)

const (
	maintenanceTaskInitializingMessage     = "initializing maintenance task"
	maintenanceTaskInitializedMessage      = "maintenance task has been initialized"
	maintenanceTaskInitializeFailedMessage = "failed to initialize maintenance task"
	maintenanceTaskStartedMessage          = "maintenance task has been started"
	maintenanceTaskCompletedMessage        = "maintenance task has been completed"

	cleanupStaleAgentsSchedule = 1 * time.Hour
	cleanupStaleAgentsId       = "cleanupStaleAgents"
)

type Cron struct {
	cmd       *cli.Command
	store     store.Store
	scheduler gocron.Scheduler
}

func NewCron(cmd *cli.Command, store store.Store) (*Cron, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return &Cron{
		cmd:       cmd,
		store:     store,
		scheduler: scheduler,
	}, nil
}

func (c *Cron) Start() {
	agentsRetention := c.cmd.String("maintenance-cleanup-agents-older-than")
	if agentsRetention != "" {
		c.setupStaleAgentsCleanup(agentsRetention)
	}
	c.scheduler.Start()
}

func (c *Cron) setupStaleAgentsCleanup(retentionStr string) {
	log.Debug().Str("task", cleanupStaleAgentsId).Msg(maintenanceTaskInitializingMessage)

	retention, err := time.ParseDuration(retentionStr)
	if err != nil {
		log.Error().Err(err).Str("task", cleanupStaleAgentsId).Msg(maintenanceTaskInitializeFailedMessage)
		return
	}

	jobDef := gocron.DurationJob(cleanupStaleAgentsSchedule)
	task := gocron.NewTask(cleanupStaleAgents, c.store, retention)
	_, err = c.scheduler.NewJob(jobDef, task)
	if err != nil {
		log.Error().Err(err).Str("task", cleanupStaleAgentsId).Msg(maintenanceTaskInitializeFailedMessage)
		return
	}

	log.Info().Str("task", cleanupStaleAgentsId).
		Str("retention", retention.String()).
		Msg(maintenanceTaskInitializedMessage)
}

func cleanupStaleAgents(store store.Store, retention time.Duration) {
	log.Debug().Str("task", cleanupStaleAgentsId).Msg(maintenanceTaskStartedMessage)

	agents, err := store.AgentList(&model.ListOptions{All: true})
	if err != nil {
		log.Error().Err(err).Str("task", cleanupStaleAgentsId).Msg("failed to get agents list")
		return
	}

	for _, agent := range agents {
		lastContacted := time.Unix(agent.LastContact, 0)
		if time.Since(lastContacted) > retention {
			log.Debug().
				Str("id", strconv.FormatInt(agent.ID, 10)).
				Str("lastContact", strconv.FormatInt(agent.LastContact, 10)).
				Msg("deleting agent")

			err = store.AgentDelete(agent)
			if err != nil {
				log.Error().Err(err).Str("task", cleanupStaleAgentsId).Msg("failed to delete agent")
				continue
			}
		}
	}

	log.Debug().Str("task", cleanupStaleAgentsId).Msg(maintenanceTaskCompletedMessage)
}
