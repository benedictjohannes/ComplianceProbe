package playbook_test

import (
	"testing"

	"github.com/benedictjohannes/crobe/playbook"
)

func TestPlaybookElevation(t *testing.T) {
	t.Run("nil playbook", func(t *testing.T) {
		var pb *playbook.Playbook
		if pb.RequiresElevation() {
			t.Error("expected false for nil playbook")
		}
		nSec, nAss, nCmds, req := pb.ElevationStats()
		if nSec != 0 || nAss != 0 || nCmds != 0 || req {
			t.Errorf("unexpected stats for nil playbook: %d, %d, %d, %v", nSec, nAss, nCmds, req)
		}
	})

	t.Run("no elevation required", func(t *testing.T) {
		pb := &playbook.Playbook{
			Sections: []playbook.Section{
				{
					Assertions: []playbook.Assertion{
						{
							Cmds: []playbook.Cmd{
								{Exec: playbook.Exec{Script: "echo 1", RequireElevation: false}},
							},
						},
					},
				},
			},
		}
		if pb.RequiresElevation() {
			t.Error("expected false when no elevation required")
		}
		nSec, nAss, nCmds, req := pb.ElevationStats()
		if nSec != 0 || nAss != 0 || nCmds != 0 || req {
			t.Errorf("unexpected stats: %d, %d, %d, %v", nSec, nAss, nCmds, req)
		}
	})

	t.Run("elevated Cmds, PreCmds, PostCmds", func(t *testing.T) {
		pb := &playbook.Playbook{
			Sections: []playbook.Section{
				{
					Assertions: []playbook.Assertion{
						{
							PreCmds: []playbook.Exec{
								{Script: "whoami", RequireElevation: true},
							},
							Cmds: []playbook.Cmd{
								{Exec: playbook.Exec{Script: "id", RequireElevation: true}},
								{Exec: playbook.Exec{Script: "echo non-elevated", RequireElevation: false}},
							},
							PostCmds: []playbook.Exec{
								{Script: "cleanup", RequireElevation: true},
							},
						},
					},
				},
				{
					Assertions: []playbook.Assertion{
						{
							Cmds: []playbook.Cmd{
								{Exec: playbook.Exec{Script: "ls", RequireElevation: false}},
							},
						},
					},
				},
			},
		}

		if !pb.RequiresElevation() {
			t.Error("expected true when elevation is required")
		}

		nSec, nAss, nCmds, req := pb.ElevationStats()
		if nSec != 1 || nAss != 1 || nCmds != 3 || !req {
			t.Errorf("got stats (%d, %d, %d, %v), want (1, 1, 3, true)", nSec, nAss, nCmds, req)
		}
	})
}
