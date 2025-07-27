package runner_test

import (
	"context"
	"testing"

	"github.com/ashep/go-app/runner"
)

func TestNew(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		runner.New(appFactory)
	})
}

type testApp struct {
}

func (a *testApp) Run(context.Context) error {
	return nil
}

type testConfig struct {
}

func (c *testConfig) Validate() error {
	return nil
}

func appFactory(*testConfig, *runner.Runtime) (*testApp, error) {
	return &testApp{}, nil
}
