//go:build wireinject
// +build wireinject

package ezjob

import (
	"github.com/google/wire"
	"github.com/jackie8tao/ezjob/internal/pkg/cluster"
	"github.com/jackie8tao/ezjob/internal/pkg/dispatcher"
	"github.com/jackie8tao/ezjob/internal/pkg/event"
	extcron2 "github.com/jackie8tao/ezjob/internal/pkg/extcron"
	server2 "github.com/jackie8tao/ezjob/internal/pkg/server"
	"github.com/jackie8tao/ezjob/internal/pkg/watcher"
	pb "github.com/jackie8tao/ezjob/proto"
)

func nodeProvider(cfg *pb.AppConfig) *cluster.Node {
	wire.Build(
		cluster.NewNode,
		newEtcdCli,
		event.NewManager,
		newEtcdCfg,
	)
	return &cluster.Node{}
}

func watcherProvider(cfg *pb.AppConfig) *watcher.Watcher {
	wire.Build(
		watcher.NewWatcher,
		newEtcdCli,
		event.NewManager,
		newEtcdCfg,
	)
	return &watcher.Watcher{}
}

func grpcServerProvider(cfg *pb.AppConfig) *server2.GrpcServer {
	wire.Build(
		server2.NewGrpcServer,
		newEtcdCli,
		newEtcdCfg,
		newGrpcCfg,
		newMysqlCfg,
		newGormDB,
	)
	return &server2.GrpcServer{}
}

func httpServerProvider(cfg *pb.AppConfig) *server2.HttpServer {
	wire.Build(
		server2.NewHttpServer,
		newHttpCfg,
	)
	return &server2.HttpServer{}
}

func schedProvider(cfg *pb.AppConfig) *extcron2.Scheduler {
	wire.Build(
		extcron2.NewScheduler,
		newEtcdCli,
		newEtcdCfg,
		event.NewManager,
		newMysqlCfg,
		newGormDB,
	)
	return &extcron2.Scheduler{}
}

func dispatcherProvider(cfg *pb.AppConfig) *dispatcher.Dispatcher {
	wire.Build(
		dispatcher.NewDispatcher,
		newEtcdCli,
		newEtcdCfg,
		newDispatcherCfg,
		event.NewManager,
		newMysqlCfg,
		newGormDB,
	)
	return &dispatcher.Dispatcher{}
}

func sentryProvider(cfg *pb.AppConfig) *extcron2.Sentry {
	wire.Build(
		extcron2.NewSentry,
		newEtcdCli,
		newEtcdCfg,
		event.NewManager,
		newMysqlCfg,
		newGormDB,
	)
	return &extcron2.Sentry{}
}
