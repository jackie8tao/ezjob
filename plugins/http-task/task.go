package httptask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	whttp "github.com/ThreeDotsLabs/watermill-http/v2/pkg/http"
	"github.com/go-chi/chi"
	pb "github.com/jackie8tao/ezjob/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Task interface {
	Name() string
	Execute(ctx context.Context, payload *pb.HttpProcPayload) error
}

type Server struct {
	cfg    Config
	task   Task
	client pb.EzJobClient
}

func NewServer(cfg Config) (*Server, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(cfg.ServerAddr, opts...)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:    cfg,
		client: pb.NewEzJobClient(conn),
	}, nil
}

func (s *Server) SetTask(task Task) {
	s.task = task
	return
}

func (s *Server) Serve(ctx context.Context) error {
	chCfg := whttp.SubscriberConfig{
		Router:               chi.NewRouter(),
		UnmarshalMessageFunc: whttp.DefaultUnmarshalMessageFunc,
	}

	sub, err := whttp.NewSubscriber(s.cfg.TaskAddr, chCfg, watermill.NewStdLogger(s.cfg.Debug, false))
	if err != nil {
		return err
	}

	ch, err := sub.Subscribe(ctx, fmt.Sprintf("/eztasks/%s", s.task.Name()))
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case msg := <-ch:
				payload := &pb.HttpProcPayload{}
				err = json.Unmarshal(msg.Payload, payload)
				if err != nil {
					continue
				}
				go s.executeTask(ctx, payload)
				msg.Ack()
			}
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			_ = sub.Close()
		}
	}()

	err = sub.StartHTTPServer()
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) executeTask(ctx context.Context, payload *pb.HttpProcPayload) {
	err := s.task.Execute(ctx, payload)
	if err != nil {
		log.Errorf("execute task error: %v", err)
	}

	err = s.reportExec(payload.TaskName, payload.ExecutionId, err)
	if err != nil {
		log.Errorf("report task error: %v", err)
	}
}

func (s *Server) reportExec(name string, execId int32, msg error) error {
	req := &pb.ReportReq{
		JobName:     name,
		Status:      pb.JobStatusSuccess,
		ExecutionId: execId,
	}
	if msg != nil {
		req.Status = pb.JobStatusFail
		req.Log = msg.Error()
	}

	_, err := s.client.Report(context.Background(), req)
	if err != nil {
		return err
	}

	return nil
}
