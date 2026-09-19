package service

import (
	"errors"
	"testing"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/dto"
)

func TestRemoteActionRequiresSecondConfirmation(t *testing.T) {
	input := dto.TransitionRequest{Status: "confirmed", ExpectedVersion: 1, Reason: "operator approval"}
	err := requireSecondConfirmation(input)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected missing confirmation to fail, got %v", err)
	}
	input.Confirmed = true
	if err := requireSecondConfirmation(input); err != nil {
		t.Fatalf("expected explicit confirmation to pass, got %v", err)
	}
}
