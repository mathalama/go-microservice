package client

import (
	"context"

	doctorpb "doctor-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DoctorGRPCClient struct {
	client doctorpb.DoctorServiceClient
}

func NewDoctorGRPCClient(conn grpc.ClientConnInterface) *DoctorGRPCClient {
	return &DoctorGRPCClient{
		client: doctorpb.NewDoctorServiceClient(conn),
	}
}

func (c *DoctorGRPCClient) DoctorExists(ctx context.Context, doctorID string) (bool, error) {
	_, err := c.client.GetDoctor(ctx, &doctorpb.GetDoctorRequest{Id: doctorID})
	if err == nil {
		return true, nil
	}

	st, ok := status.FromError(err)
	if ok && st.Code() == codes.NotFound {
		return false, nil
	}

	return false, err
}
