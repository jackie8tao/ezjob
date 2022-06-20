//go:build wireinject
// +build wireinject

package wireutil

import (
	"github.com/google/wire"
	"github.com/jackie8tao/ezjob/pkg/cluster"
	"github.com/jackie8tao/ezjob/pkg/dispatcher"
	"github.com/jackie8tao/ezjob/pkg/event"
	"github.com/jackie8tao/ezjob/pkg/extcron"
	"github.com/jackie8tao/ezjob/pkg/server"
	"github.com/jackie8tao/ezjob/pkg/watcher"
	pb "github.com/jackie8tao/ezjob/proto"
)

func NodeProvider(cfg *pb.AppConfig) *cluster.Node {
	wire.Build(
		cluster.NewNode,
		newEtcdCli,
		event.NewManager,
		newEtcdCfg,
	)
	return &cluster.Node{}
}

func WatcherProvider(cfg *pb.AppConfig) *watcher.Watcher {
	wire.Build(
		watcher.NewWatcher,
		newEtcdCli,
		event.NewManager,
		newEtcdCfg,
	)
	return &watcher.Watcher{}
}

func GrpcServerProvider(cfg *pb.AppConfig) *server.GrpcServer {
	wire.Build(
		server.NewGrpcServer,
		newEtcdCli,
		newEtcdCfg,
		newGrpcCfg,
		newMysqlCfg,
		newGormDB,
	)
	return &server.GrpcServer{}
}

func HttpServerProvider(cfg *pb.AppConfig) *server.HttpServer {
	wire.Build(
		server.NewHttpServer,
		newHttpCfg,
	)
	return &server.HttpServer{}
}

func SchedProvider(cfg *pb.AppConfig) *extcron.Scheduler {
	wire.Build(
		extcron.NewScheduler,
		newEtcdCli,
		newEtcdCfg,
		event.NewManager,
		newMysqlCfg,
		newGormDB,
	)
	return &extcron.Scheduler{}
}

func DispatcherProvider(cfg *pb.AppConfig) *dispatcher.Dispatcher {
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

func SentryProvider(cfg *pb.AppConfig) *extcron.Sentry {
	wire.Build(
		extcron.NewSentry,
		newEtcdCli,
		newEtcdCfg,
		event.NewManager,
		newMysqlCfg,
		newGormDB,
	)
	return &extcron.Sentry{}
}
