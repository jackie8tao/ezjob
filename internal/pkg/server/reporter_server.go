package server

import (
	"context"
	"errors"
	"net"

	"github.com/jackie8tao/ezjob/internal/model"
	pb "github.com/jackie8tao/ezjob/proto"
	log "github.com/sirupsen/logrus"
	etcdcli "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type ReporterServer struct {
	pb.UnimplementedReporterServer
	cfg    *pb.GrpcConfig
	cli    *etcdcli.Client
	logger log.FieldLogger
	db     *gorm.DB
	srv    *grpc.Server
}

func NewReporterServer(cfg *pb.GrpcConfig, cli *etcdcli.Client, db *gorm.DB) *ReporterServer {
	return &ReporterServer{
		cli:    cli,
		cfg:    cfg,
		logger: log.WithField("module", "grpc"),
		db:     db,
	}
}

func (s *ReporterServer) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	lis, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption
	s.srv = grpc.NewServer(opts...)
	pb.RegisterReporterServer(s.srv, s)

	go func() {
		if err := s.srv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	s.logger.Debug("grpc booted")

	return nil
}

func (s *ReporterServer) Close() error {
	err := s.cli.Close()
	if err != nil {
		return err
	}

	s.srv.Stop()

	s.logger.Debug("grpc exited")

	return nil
}

func (s *ReporterServer) Report(ctx context.Context, req *pb.ReportReq) (rsp *emptypb.Empty, err error) {
	err = req.Validate()
	if err != nil {
		return
	}

	var exec model.Execution
	err = s.db.Where("id = ?", req.ExecutionId).First(&exec).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			err = errors.New("job execution is null")
			return
		}
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	switch req.Status {
	case pb.JobStatusSuccess:
		exec.Status = pb.JobStatusSuccess
	case pb.JobStatusFail:
		exec.Status = pb.JobStatusFail
	default:
	}
	exec.Result = req.Log

	err = s.db.Save(&exec).Error
	if err != nil {
		return
	}

	rsp = &emptypb.Empty{}
	return
}
