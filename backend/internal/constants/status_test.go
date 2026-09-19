package constants

import "testing"

func TestSolarSiteTransitionGraph(t *testing.T) {
	if !CanTransition(SolarSiteTransitions, "online", "limited") {
		t.Fatalf("expected online -> limited transition to be allowed")
	}
	if CanTransition(SolarSiteTransitions, "online", "unknown") {
		t.Fatal("unknown status must never be accepted")
	}
}
