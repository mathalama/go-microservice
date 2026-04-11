package grpc

import (
	"context"
	"errors"
	"time"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	appointmentpb "appointment-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppointmentServer struct {
	appointmentpb.UnimplementedAppointmentServiceServer
	uc usecase.AppointmentUseCase
}

func NewAppointmentServer(uc usecase.AppointmentUseCase) *AppointmentServer {
	return &AppointmentServer{uc: uc}
}

func (s *AppointmentServer) Register(server *grpc.Server) {
	appointmentpb.RegisterAppointmentServiceServer(server, s)
}

func (s *AppointmentServer) CreateAppointment(ctx context.Context, req *appointmentpb.CreateAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := s.uc.CreateAppointment(ctx, req.GetTitle(), req.GetDescription(), req.GetDoctorId())
	if err != nil {
		return nil, mapAppointmentError(err)
	}
	return toAppointmentResponse(appointment), nil
}

func (s *AppointmentServer) GetAppointment(ctx context.Context, req *appointmentpb.GetAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := s.uc.GetAppointment(ctx, req.GetId())
	if err != nil {
		return nil, mapAppointmentError(err)
	}
	return toAppointmentResponse(appointment), nil
}

func (s *AppointmentServer) ListAppointments(ctx context.Context, req *appointmentpb.ListAppointmentsRequest) (*appointmentpb.ListAppointmentsResponse, error) {
	appointments, err := s.uc.ListAppointments(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list appointments")
	}

	response := &appointmentpb.ListAppointmentsResponse{
		Appointments: make([]*appointmentpb.AppointmentResponse, 0, len(appointments)),
	}
	for _, appointment := range appointments {
		response.Appointments = append(response.Appointments, toAppointmentResponse(appointment))
	}

	return response, nil
}

func (s *AppointmentServer) UpdateAppointmentStatus(ctx context.Context, req *appointmentpb.UpdateStatusRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := s.uc.UpdateStatus(ctx, req.GetId(), model.Status(req.GetStatus()))
	if err != nil {
		return nil, mapAppointmentError(err)
	}
	return toAppointmentResponse(appointment), nil
}

func toAppointmentResponse(appointment model.Appointment) *appointmentpb.AppointmentResponse {
	return &appointmentpb.AppointmentResponse{
		Id:          appointment.ID,
		Title:       appointment.Title,
		Description: appointment.Description,
		DoctorId:    appointment.DoctorID,
		Status:      string(appointment.Status),
		CreatedAt:   appointment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   appointment.UpdatedAt.Format(time.RFC3339),
	}
}

func mapAppointmentError(err error) error {
	switch {
	case errors.Is(err, model.ErrAppointmentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, model.ErrDoctorNotFound):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, model.ErrDependencyUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, model.ErrInvalidStatus), errors.Is(err, model.ErrForbiddenStatusTransit):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.InvalidArgument, err.Error())
	}
}
