package playbook

// ElevationStats analyzes the playbook and returns counts of sections, assertions,
// and commands (including PreCmds, Cmds, and PostCmds) that require elevated privileges,
// along with a boolean indicating if any command requires elevation.
func (p *Playbook) ElevationStats() (nSections, nAssertions, nCmds int, requireElevation bool) {
	if p == nil {
		return 0, 0, 0, false
	}

	secSet := make(map[int]struct{})
	assSet := make(map[[2]int]struct{})

	for sIdx, sec := range p.Sections {
		for aIdx, ass := range sec.Assertions {
			key := [2]int{sIdx, aIdx}
			for _, exec := range ass.PreCmds {
				if exec.RequireElevation {
					nCmds++
					secSet[sIdx] = struct{}{}
					assSet[key] = struct{}{}
				}
			}
			for _, cmd := range ass.Cmds {
				if cmd.Exec.RequireElevation {
					nCmds++
					secSet[sIdx] = struct{}{}
					assSet[key] = struct{}{}
				}
			}
			for _, exec := range ass.PostCmds {
				if exec.RequireElevation {
					nCmds++
					secSet[sIdx] = struct{}{}
					assSet[key] = struct{}{}
				}
			}
		}
	}

	return len(secSet), len(assSet), nCmds, nCmds > 0
}

// RequiresElevation returns true if any command within the playbook requires elevated privileges.
func (p *Playbook) RequiresElevation() bool {
	if p == nil {
		return false
	}
	_, _, _, req := p.ElevationStats()
	return req
}
