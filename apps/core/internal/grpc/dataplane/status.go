package dataplane

import (
	"context"

	"github.com/hydradns/hydra-core/internal/dnsengine"
	pb "github.com/hydradns/hydra-core/internal/gen/proto/hydradns/v1"
)

type StatusService struct {
	pb.UnimplementedDataPlaneStatusServiceServer
	engine *dnsengine.Engine
}

func NewStatusService(engine *dnsengine.Engine) *StatusService {
	return &StatusService{
		engine: engine,
	}
}

func (s *StatusService) GetStatus(ctx context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	st := s.engine.Status()

	return &pb.StatusResponse{
		Running:          st.Running,
		AcceptingQueries: st.AcceptingQueries,
		PolicyEnabled:    st.PolicyEnabled,
		LastError:        st.LastError,
	}, nil
}

func (s *StatusService) SetAcceptQueries(
	ctx context.Context,
	req *pb.SetAcceptQueriesRequest,
) (*pb.SetAcceptQueriesResponse, error) {

	// Apply desired state to the engine.
	s.engine.SetAcceptQueries(req.Enabled)

	return &pb.SetAcceptQueriesResponse{
		Ok: true,
	}, nil
}
