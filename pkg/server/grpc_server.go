package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/jackie8tao/ezjob/internal/model"
	pb "github.com/jackie8tao/ezjob/proto"
	"github.com/jackie8tao/ezjob/utils/jobutil"
	log "github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type GrpcServer struct {
	pb.UnimplementedEzJobServer
	cfg    *pb.GrpcConfig
	cli    *clientv3.Client
	logger log.FieldLogger
	db     *gorm.DB
}

func NewGrpcServer(cfg *pb.GrpcConfig, cli *clientv3.Client, db *gorm.DB) *GrpcServer {
	return &GrpcServer{
		cli:    cli,
		cfg:    cfg,
		logger: log.WithField("module", "grpc"),
		db:     db,
	}
}

func (s *GrpcServer) Boot(ctx context.Context) error {
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
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterEzJobServer(grpcServer, s)

	go func() {
		innerErr := grpcServer.Serve(lis)
		if innerErr != nil {
			panic(err)
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			grpcServer.Stop()
			s.logger.Debug("grpc stopped")
		}
	}()

	s.logger.Debug("grpc booted")

	return nil
}

func (s *GrpcServer) Close() error {
	err := s.cli.Close()
	if err != nil {
		return err
	}

	s.logger.Debug("grpc exited")

	return nil
}

func (s *GrpcServer) CreateJob(ctx context.Context, job *pb.Job) (rsp *emptypb.Empty, err error) {
	err = job.Validate()
	if err != nil {
		return
	}

	data, err := json.Marshal(job)
	if err != nil {
		return
	}
	_, err = s.cli.Put(ctx, jobutil.JobKey(job.Name), string(data))
	if err != nil {
		return
	}

	rsp = &emptypb.Empty{}

	return
}

func (s *GrpcServer) ListJob(ctx context.Context, req *pb.ListJobReq) (rsp *pb.ListJobRsp, err error) {
	var opts []clientv3.OpOption
	key := jobutil.JobKey(req.Name)
	if req.Name == "" {
		opts = append(opts, clientv3.WithPrefix())
		opts = append(opts, clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	}

	ret, err := s.cli.Get(ctx, key, opts...)
	if err != nil {
		return
	}

	jobs := make([]*pb.Job, 0)
	for _, v := range ret.Kvs {
		item := &pb.Job{}
		err = json.Unmarshal(v.Value, item)
		if err != nil {
			return
		}
		jobs = append(jobs, item)
	}

	rsp = &pb.ListJobRsp{
		Tasks: jobs,
	}

	return
}

func (s *GrpcServer) DelJob(ctx context.Context, req *pb.DelJobReq) (rsp *emptypb.Empty, err error) {
	err = req.Validate()
	if err != nil {
		return
	}

	_, err = s.cli.Delete(ctx, jobutil.JobKey(req.Name))
	if err != nil {
		return
	}

	rsp = &emptypb.Empty{}

	return
}

func (s *GrpcServer) RunJob(ctx context.Context, req *pb.RunJobReq) (rsp *emptypb.Empty, err error) {
	err = req.Validate()
	if err != nil {
		return
	}

	tasks, err := s.cli.Get(ctx, jobutil.JobKey(req.Name))
	if err != nil {
		return
	}

	if len(tasks.Kvs) <= 0 {
		err = fmt.Errorf("job %s is not registered", req.Name)
		return
	}

	job := &pb.Job{}
	err = json.Unmarshal(tasks.Kvs[0].Value, job)
	if err != nil {
		return
	}

	switch job.Proc.Type {
	case pb.ProcType_HTTP:
		job.Proc.HttpProc.Payload = req.Payload
	default:
	}

	data, err := json.Marshal(job)
	if err != nil {
		return
	}
	_, err = s.cli.Put(ctx, jobutil.TriggerKey(req.Name), string(data))
	if err != nil {
		return
	}

	rsp = &emptypb.Empty{}

	return
}

func (s *GrpcServer) Report(ctx context.Context, req *pb.ReportReq) (rsp *emptypb.Empty, err error) {
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
