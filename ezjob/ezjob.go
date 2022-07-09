// Package ezjob api服务
package ezjob

import (
	"context"
	"encoding/json"
	"io/ioutil"

	"github.com/jackie8tao/ezjob/internal/pkg/cluster"
	"github.com/jackie8tao/ezjob/internal/pkg/dispatcher"
	"github.com/jackie8tao/ezjob/internal/pkg/event"
	"github.com/jackie8tao/ezjob/internal/pkg/extcron"
	server2 "github.com/jackie8tao/ezjob/internal/pkg/server"
	"github.com/jackie8tao/ezjob/internal/pkg/watcher"
	pb "github.com/jackie8tao/ezjob/proto"
	log "github.com/sirupsen/logrus"
)

type EzJob struct {
	evtMgr     *event.Manager
	node       *cluster.Node
	scheduler  *extcron.Scheduler
	dispatcher *dispatcher.Dispatcher
	grpcSrv    *server2.GrpcServer
	httpSrv    *server2.HttpServer
	watcher    *watcher.Watcher
	sentry     *extcron.Sentry
}

func NewEzJob(cfg *pb.AppConfig) (*EzJob, error) {
	log.SetLevel(log.DebugLevel)
	obj := &EzJob{
		evtMgr:     event.NewManager(),
		node:       nodeProvider(cfg),
		scheduler:  schedProvider(cfg),
		dispatcher: dispatcherProvider(cfg),
		grpcSrv:    grpcServerProvider(cfg),
		watcher:    watcherProvider(cfg),
		httpSrv:    httpServerProvider(cfg),
		sentry:     sentryProvider(cfg),
	}

	return obj, nil
}

func LoadCfg(file string) (*pb.AppConfig, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cfg := &pb.AppConfig{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (e *EzJob) Serve(ctx context.Context) error {
	var err error
	for _, m := range e.modules() {
		err = m.Boot(ctx)
		if err != nil {
			return err
		}
	}

	err = e.evtMgr.Boot(ctx)
	if err != nil {
		return err
	}

	e.node.StartElection()

	return nil
}

func (e *EzJob) Close() error {
	var err error
	for _, m := range e.modules() {
		err = m.Close()
		if err != nil {
			return err
		}
	}

	err = e.evtMgr.Close()
	if err != nil {
		return err
	}

	return nil
}

func (e *EzJob) modules() []pb.Module {
	return []pb.Module{
		e.node,
		e.watcher,
		e.grpcSrv,
		e.httpSrv,
		e.scheduler,
		e.dispatcher,
		e.sentry,
	}
}
