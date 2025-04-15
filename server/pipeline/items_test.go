package pipeline

import (
	"github.com/stretchr/testify/assert"
	forge_types "go.woodpecker-ci.org/woodpecker/v2/server/forge/types"
	"testing"

	"go.woodpecker-ci.org/woodpecker/v2/pipeline/backend/types"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	sharedPipeline "go.woodpecker-ci.org/woodpecker/v2/server/pipeline/stepbuilder"
)

func TestSetPipelineStepsOnPipeline(t *testing.T) {
	t.Parallel()

	pipeline := &model.Pipeline{
		ID:    1,
		Event: model.EventPush,
	}

	pipelineItems := []*sharedPipeline.Item{{
		Workflow: &model.Workflow{
			PID: 1,
		},
		Config: &types.Config{
			Stages: []*types.Stage{
				{
					Steps: []*types.Step{
						{
							Name: "clone",
						},
					},
				},
				{
					Steps: []*types.Step{
						{
							Name: "step",
						},
					},
				},
			},
		},
	}}
	pipeline = setPipelineStepsOnPipeline(pipeline, pipelineItems)
	if len(pipeline.Workflows) != 1 {
		t.Fatal("Should generate three in total")
	}
	if pipeline.Workflows[0].PipelineID != 1 {
		t.Fatal("Should set workflow's pipeline ID")
	}
	if pipeline.Workflows[0].Children[0].PPID != 1 {
		t.Fatal("Should set step PPID")
	}
}

func TestJsonnet(t *testing.T) {
	jsonnetPipeline := []byte(`
		{
			steps: {
				hello: {
					image: "alpine",
					commands: [
						std.join(" ", ["echo", "Hello", self.image, "!"]),
						'echo The repo name is %s and event is %s' % [ std.extVar('CI_REPO_NAME'), std.extVar('CI_PIPELINE_EVENT') ],
					]
				},
			},
		}
	`)
	config := forge_types.FileMeta{
		Name: "woodpecker.jsonnet",
		Data: jsonnetPipeline,
	}
	configs := []*forge_types.FileMeta{&config}

	envs := map[string]string{
		"CI_REPO_NAME":      "test-repo",
		"CI_PIPELINE_EVENT": "push",
	}

	err := evaluateJsonnet(configs, envs)
	assert.NoError(t, err)

	t.Log(string(config.Data))
}
