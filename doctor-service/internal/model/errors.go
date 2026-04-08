package model

import "errors"

var (
	ErrDoctorNotFound = errors.New("doctor not found")
	ErrEmailExists    = errors.New("doctor with this email already exists")
)
