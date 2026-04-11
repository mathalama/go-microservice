package grpc

import (
	"context"
	"errors"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
	doctorpb "doctor-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DoctorServer struct {
	doctorpb.UnimplementedDoctorServiceServer
	uc usecase.DoctorUsecase
}

func NewDoctorServer(uc usecase.DoctorUsecase) *DoctorServer {
	return &DoctorServer{uc: uc}
}

func (s *DoctorServer) Register(server *grpc.Server) {
	doctorpb.RegisterDoctorServiceServer(server, s)
}

func (s *DoctorServer) CreateDoctor(ctx context.Context, req *doctorpb.CreateDoctorRequest) (*doctorpb.DoctorResponse, error) {
	doctor, err := s.uc.CreateDoctor(ctx, req.GetFullName(), req.GetSpecialization(), req.GetEmail())
	if err != nil {
		return nil, mapDoctorError(err)
	}
	return toDoctorResponse(doctor), nil
}

func (s *DoctorServer) GetDoctor(ctx context.Context, req *doctorpb.GetDoctorRequest) (*doctorpb.DoctorResponse, error) {
	doctor, err := s.uc.GetDoctor(ctx, req.GetId())
	if err != nil {
		return nil, mapDoctorError(err)
	}
	return toDoctorResponse(doctor), nil
}

func (s *DoctorServer) ListDoctors(ctx context.Context, req *doctorpb.ListDoctorsRequest) (*doctorpb.ListDoctorsResponse, error) {
	doctors, err := s.uc.ListDoctors(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list doctors")
	}

	response := &doctorpb.ListDoctorsResponse{
		Doctors: make([]*doctorpb.DoctorResponse, 0, len(doctors)),
	}
	for _, doctor := range doctors {
		response.Doctors = append(response.Doctors, toDoctorResponse(doctor))
	}
	return response, nil
}

func toDoctorResponse(doctor model.Doctor) *doctorpb.DoctorResponse {
	return &doctorpb.DoctorResponse{
		Id:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}
}

func mapDoctorError(err error) error {
	switch {
	case errors.Is(err, model.ErrDoctorNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, model.ErrEmailExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.InvalidArgument, err.Error())
	}
}
