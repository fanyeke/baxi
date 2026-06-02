package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepOutput_Fields(t *testing.T) {
	output := &StepOutput{InputCount: 10, OutputCount: 8}
	assert.Equal(t, int64(10), output.InputCount)
	assert.Equal(t, int64(8), output.OutputCount)
}

func TestStepOutput_ZeroValues(t *testing.T) {
	output := &StepOutput{}
	assert.Equal(t, int64(0), output.InputCount)
	assert.Equal(t, int64(0), output.OutputCount)
}

func TestStepOutput_NegativeValues(t *testing.T) {
	output := &StepOutput{InputCount: -1, OutputCount: -5}
	assert.Equal(t, int64(-1), output.InputCount)
	assert.Equal(t, int64(-5), output.OutputCount)
}

func TestStepInput_Fields(t *testing.T) {
	input := StepInput{
		RunID:   "run-123",
		Logger:  nil,
		DataDir: "/tmp/data",
	}
	assert.Equal(t, "run-123", input.RunID)
	assert.Nil(t, input.Logger)
	assert.Equal(t, "/tmp/data", input.DataDir)
}

func TestRunInput_Fields(t *testing.T) {
	input := RunInput{
		RunType: "full",
		Mode:    "manual",
		DataDir: "/data",
	}
	assert.Equal(t, "full", input.RunType)
	assert.Equal(t, "manual", input.Mode)
	assert.Equal(t, "/data", input.DataDir)
}

func TestMockStep_Name(t *testing.T) {
	step := &mockStep{name: "test-step", succ: true}
	assert.Equal(t, "test-step", step.Name())
}

func TestMockStep_RunSuccess(t *testing.T) {
	step := &mockStep{name: "test-step", succ: true}
	output, err := step.Run(t.Context(), nil, StepInput{})
	assert.NoError(t, err)
	assert.Equal(t, int64(5), output.InputCount)
	assert.Equal(t, int64(5), output.OutputCount)
}

func TestMockStep_RunFailure(t *testing.T) {
	step := &mockStep{name: "failing-step", succ: false}
	output, err := step.Run(t.Context(), nil, StepInput{})
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failing-step")
}

func TestStepSpy_CalledTracking(t *testing.T) {
	spy := &stepSpy{mockStep: &mockStep{name: "spy-step", succ: true}}
	assert.False(t, spy.called)

	output, err := spy.Run(t.Context(), nil, StepInput{})
	assert.NoError(t, err)
	assert.True(t, spy.called)
	assert.NotNil(t, output)
}

func TestStepSpy_Failure(t *testing.T) {
	spy := &stepSpy{mockStep: &mockStep{name: "spy-fail", succ: false}}
	_, err := spy.Run(t.Context(), nil, StepInput{})
	assert.Error(t, err)
	assert.True(t, spy.called)
}

func TestRunnerStepsField(t *testing.T) {
	runner := &Runner{
		Steps: []Step{
			&mockStep{name: "a", succ: true},
			&mockStep{name: "b", succ: true},
		},
	}
	assert.Len(t, runner.Steps, 2)
}

func TestRunnerEmptySteps(t *testing.T) {
	runner := &Runner{
		Steps: []Step{},
	}
	assert.Empty(t, runner.Steps)
}
