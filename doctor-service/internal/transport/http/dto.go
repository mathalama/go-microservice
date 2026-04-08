package http

import "doctor-service/internal/model"

type DoctorResponse struct {
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

func ToDoctorResponse(d model.Doctor) DoctorResponse {
	return DoctorResponse{
		ID:             d.ID,
		FullName:       d.FullName,
		Specialization: d.Specialization,
		Email:          d.Email,
	}
}
