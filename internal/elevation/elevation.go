package elevation

import (
	"fmt"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
)

func SetupElevation(config *playbook.Playbook) (func(), error) {
	if config == nil || !config.RequiresElevation() || executor.GetElevatedExecutionRunner() != nil {
		return func() {}, nil
	}

	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize elevated runner: %w", err)
	}

	executor.SetElevatedExecutionRunner(client)

	cleanup := func() {
		_ = client.Close()
		executor.SetElevatedExecutionRunner(nil)
	}

	return cleanup, nil
}
