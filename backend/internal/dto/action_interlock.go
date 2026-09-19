package dto

import "github.com/blueship581/solar-inverter-incident-control/backend/internal/model"

// ActionInterlockView is the fault-interlock snapshot shown before and after a
// remote confirmation. FailureReason is empty when all safety checks pass.
type ActionInterlockView struct {
	Action        model.MitigationAction `json:"action"`
	Fault         *model.FaultEvent      `json:"fault"`
	Inverter      *model.InverterUnit    `json:"inverter"`
	FailureReason string                 `json:"failureReason,omitempty"`
	CanConfirm    bool                   `json:"canConfirm"`
}
