package domain

// CheckState is the outcome of a build or check that ran for a pull request.
type CheckState string

const (
	CheckStateNone    CheckState = "none"
	CheckStatePending CheckState = "pending"
	CheckStatePassing CheckState = "passing"
	CheckStateFailing CheckState = "failing"
)

// checkStatePriority decides the overall state: one failing check fails the pull request, and
// one still running keeps it pending.
var checkStatePriority = map[CheckState]int{
	CheckStateFailing: 3,
	CheckStatePending: 2,
	CheckStatePassing: 1,
	CheckStateNone:    0,
}

type Check struct {
	Name  string
	State CheckState
}

// Checks are the latest builds or checks of a pull request and their overall state.
type Checks struct {
	State  CheckState
	Checks []Check
}

func SummarizeChecks(checks []Check) Checks {
	summary := Checks{State: CheckStateNone, Checks: checks}
	for _, check := range checks {
		if checkStatePriority[check.State] > checkStatePriority[summary.State] {
			summary.State = check.State
		}
	}
	return summary
}

// Failing names the checks that failed.
func (c Checks) Failing() []string {
	var names []string
	for _, check := range c.Checks {
		if check.State == CheckStateFailing {
			names = append(names, check.Name)
		}
	}
	return names
}
