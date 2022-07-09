//go:build wireinject
// +build wireinject

package ezjob

import (
	"github.com/google/wire"
	"github.com/jackie8tao/ezjob/internal/pkg/cluster"
	"github.com/jackie8tao/ezjob/internal/pkg/dispatcher"
	"github.com/jackie8tao/ezjob/internal/pkg/event"
	"github.com/jackie8tao/ezjob/internal/pkg/extcron"
	"github.com/jackie8tao/ezjob/internal/pkg/server"
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

func reporterServerProvider(cfg *pb.AppConfig) *server.ReporterServer {
	wire.Build(
		server.NewReporterServer,
		newEtcdCli,
		newEtcdCfg,
		newGrpcCfg,
		newMysqlCfg,
		newGormDB,
	)
	return &server.ReporterServer{}
}

func adminServerProvider(cfg *pb.AppConfig) *server.AdminServer {
	wire.Build(
		server.NewAdminServer,
		newEtcdCli,
		newEtcdCfg,
		newHttpCfg,
	)
	return &server.AdminServer{}
}

func schedProvider(cfg *pb.AppConfig) *extcron.Scheduler {
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

func sentryProvider(cfg *pb.AppConfig) *extcron.Sentry {
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
