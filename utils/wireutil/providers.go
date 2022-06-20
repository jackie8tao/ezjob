package wireutil

import (
	"fmt"

	pb "github.com/jackie8tao/ezjob/proto"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newEtcdCfg(cfg *pb.AppConfig) *pb.EtcdConfig {
	if cfg == nil {
		panic("etcd config is empty")
	}
	return cfg.Etcd
}

func newGrpcCfg(cfg *pb.AppConfig) *pb.GrpcConfig {
	if cfg.Srv == nil || cfg.Srv.GrpcCfg == nil {
		panic("grpc config is empty")
	}

	return cfg.Srv.GrpcCfg
}

func newDispatcherCfg(cfg *pb.AppConfig) *pb.DispatcherConfig {
	if cfg.Dispatcher == nil {
		panic("dispatcher config is empty")
	}

	return cfg.Dispatcher
}

func newHttpCfg(cfg *pb.AppConfig) *pb.HttpConfig {
	if cfg.Srv == nil || cfg.Srv.HttpCfg == nil {
		panic("http config is empty")
	}

	return cfg.Srv.HttpCfg
}

func newMysqlCfg(cfg *pb.AppConfig) *pb.MysqlConfig {
	if cfg.Mysql == nil {
		panic("mysql config is empty")
	}

	return cfg.Mysql
}

func newEtcdCli(cfg *pb.EtcdConfig) *clientv3.Client {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: cfg.Endpoints,
		Username:  cfg.User,
		Password:  cfg.Password,
	})
	if err != nil {
		panic(fmt.Errorf("init etcd client error: %v", err))
	}

	return cli
}

func newGormDB(cfg *pb.MysqlConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Addr,
		cfg.Db,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("init gorm error:%v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("init gorm error:%v", err))
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)

	return db
}
