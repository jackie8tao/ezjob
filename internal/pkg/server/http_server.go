package server

import (
	"context"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/jackie8tao/ezjob/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HttpServer struct {
	cfg    *pb.HttpConfig
	logger log.FieldLogger
}

func NewHttpServer(cfg *pb.HttpConfig) *HttpServer {
	return &HttpServer{
		cfg:    cfg,
		logger: log.WithField("module", "http"),
	}
}

func (h *HttpServer) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pb.RegisterEzJobHandlerFromEndpoint(context.Background(), mux, h.cfg.Upstream, opts)
	if err != nil {
		return err
	}

	server := http.Server{
		Addr:    h.cfg.Addr,
		Handler: mux,
	}

	go func() {
		innerErr := server.ListenAndServe()
		if innerErr != nil && innerErr != http.ErrServerClosed {
			panic(innerErr)
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			innerCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = server.Shutdown(innerCtx)
		}
	}()

	h.logger.Debug("http booted")

	return nil
}

func (h *HttpServer) Close() error {
	h.logger.Debug("http exited")
	return nil
}
