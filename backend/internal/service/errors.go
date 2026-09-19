package service

import "errors"

var (
	ErrInvalidTransition     = errors.New("requested status transition is not allowed")
	ErrInvalidInput          = errors.New("business input validation failed")
	ErrUnauthorized          = errors.New("invalid username or password")
	ErrInactiveUser          = errors.New("user account is inactive")
	ErrInterlockLink         = errors.New("linked fault event or inverter was not found for the action related code and facility")
	ErrInterlockPrecondition = errors.New("fault interlock precondition is not satisfied")
)
