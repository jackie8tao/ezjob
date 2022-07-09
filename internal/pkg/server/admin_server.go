package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	pb "github.com/jackie8tao/ezjob/proto"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	etcdcli "go.etcd.io/etcd/client/v3"
)

//AdminServer admin server
type AdminServer struct {
	cli    *etcdcli.Client
	engine *echo.Echo
	logger log.FieldLogger
	cfg    *pb.HttpConfig
}

func NewAdminServer(cfg *pb.HttpConfig, cli *etcdcli.Client) *AdminServer {
	return &AdminServer{
		cli:    cli,
		engine: echo.New(),
		cfg:    cfg,
		logger: log.WithField("module", "admin"),
	}
}

func (a *AdminServer) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	a.engine.GET("/api/jobs", a.ListJobs)
	a.engine.POST("/api/jobs/:name", a.CreateJob)
	a.engine.DELETE("/api/jobs/:name", a.DeleteJob)
	a.engine.POST("/api/jobs/:name/trigger", a.RunJob)

	go func() {
		err := a.engine.Start(a.cfg.Addr)
		if err != nil {
			panic(err)
		}
	}()

	a.logger.Debug("admin booted")

	return nil
}

func (a *AdminServer) Close() error {
	//TODO implement me
	panic("implement me")
}

//ListJobs list all jobs
func (a *AdminServer) ListJobs(c echo.Context) (err error) {
	req := &pb.ListJobReq{}
	if err = c.Bind(req); err != nil {
		return
	}
	if err = req.Validate(); err != nil {
		return
	}

	var opts []etcdcli.OpOption
	key := pb.GenJobKey(req.Name)
	if req.Name == "" {
		opts = append(opts, etcdcli.WithPrefix())
		opts = append(opts, etcdcli.WithSort(etcdcli.SortByKey, etcdcli.SortDescend))
	}

	ret, err := a.cli.Get(context.TODO(), key, opts...)
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

	err = c.JSON(http.StatusOK, &pb.ListJobRsp{
		Tasks: jobs,
	})

	return
}

//CreateJob create a new job
func (a *AdminServer) CreateJob(ctx echo.Context) (err error) {
	job := &pb.Job{}
	if err = ctx.Bind(job); err != nil {
		return
	}
	if err = job.Validate(); err != nil {
		return
	}

	data, err := json.Marshal(job)
	if err != nil {
		return
	}
	_, err = a.cli.Put(nil, pb.GenJobKey(job.Name), string(data))
	if err != nil {
		return
	}

	err = ctx.JSON(http.StatusOK, job)
	return
}

//DeleteJob delete a job
func (a *AdminServer) DeleteJob(ctx echo.Context) (err error) {
	req := &pb.DelJobReq{}
	if err = ctx.Bind(req); err != nil {
		return
	}
	if err = req.Validate(); err != nil {
		return
	}

	_, err = a.cli.Delete(context.TODO(), pb.GenJobKey(req.Name))
	if err != nil {
		return
	}

	return
}

//RunJob trigger a job with parameters
func (a *AdminServer) RunJob(ctx echo.Context) (err error) {
	req := &pb.RunJobReq{}
	if err = ctx.Bind(req); err != nil {
		return
	}
	if err = req.Validate(); err != nil {
		return
	}

	tasks, err := a.cli.Get(context.TODO(), pb.GenJobKey(req.Name))
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
	_, err = a.cli.Put(context.TODO(), pb.GenTriggerKey(req.Name), string(data))
	if err != nil {
		return
	}

	return
}
