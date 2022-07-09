package server

import (
	"context"
	"encoding/json"
	"net/http"

	pb "github.com/jackie8tao/ezjob/proto"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	etcdcli "go.etcd.io/etcd/client/v3"
)

//AdminServer admin server
type AdminServer struct {
	cli    *etcdcli.Client
	engine *echo.Echo
	logger logrus.FieldLogger
}

func NewAdminServer() *AdminServer {
	return &AdminServer{
		cli:    nil,
		engine: echo.New(),
	}
}

func (a *AdminServer) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	a.engine.POST("/api/jobs/:name", a.CreateJob)

	return nil
}

func (a *AdminServer) Close() error {
	//TODO implement me
	panic("implement me")
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
