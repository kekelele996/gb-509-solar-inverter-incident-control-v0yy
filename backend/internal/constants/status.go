package constants

// Shared status values are mirrored in frontend/src/types/status.ts. Keeping
// the lists explicit makes state-machine drift visible during code review.

type InverterState string

const (
	InverterStateOnline   InverterState = "online"
	InverterStateWarning  InverterState = "warning"
	InverterStateTripped  InverterState = "tripped"
	InverterStateIsolated InverterState = "isolated"
)

var AllInverterState = []string{"online", "warning", "tripped", "isolated"}

type FaultState string

const (
	FaultStateOpen         FaultState = "open"
	FaultStateAcknowledged FaultState = "acknowledged"
	FaultStateMitigated    FaultState = "mitigated"
	FaultStateClosed       FaultState = "closed"
)

var AllFaultState = []string{"open", "acknowledged", "mitigated", "closed"}

var SolarSiteTransitions = map[string]map[string]bool{
	"online":      {"limited": true, "offline": true},
	"limited":     {"offline": true, "maintenance": true, "online": true},
	"offline":     {"maintenance": true, "limited": true},
	"maintenance": {"offline": true},
}

var InverterUnitTransitions = map[string]map[string]bool{
	"online":   {"warning": true, "tripped": true},
	"warning":  {"tripped": true, "isolated": true, "online": true},
	"tripped":  {"isolated": true, "warning": true},
	"isolated": {"tripped": true},
}

var FaultEventTransitions = map[string]map[string]bool{
	"open":         {"acknowledged": true, "mitigated": true},
	"acknowledged": {"mitigated": true, "closed": true, "open": true},
	"mitigated":    {"closed": true, "acknowledged": true},
	"closed":       {"mitigated": true},
}

var MitigationActionTransitions = map[string]map[string]bool{
	"draft":     {"confirmed": true, "executing": true},
	"confirmed": {"executing": true, "completed": true, "draft": true},
	"executing": {"completed": true, "failed": true, "confirmed": true},
	"completed": {"failed": true, "executing": true},
	"failed":    {"completed": true},
}

func CanTransition(graph map[string]map[string]bool, from, to string) bool {
	targets, exists := graph[from]
	return exists && targets[to]
}
