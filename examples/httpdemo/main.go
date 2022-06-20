package main

import (
	"context"
	"fmt"
	"time"

	httptask "github.com/jackie8tao/ezjob/plugins/http-task"
	pb "github.com/jackie8tao/ezjob/proto"
)

type Hello struct{}

func (h *Hello) Name() string {
	return "hello"
}

func (h *Hello) Execute(ctx context.Context, payload *pb.HttpProcPayload) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	time.Sleep(5 * time.Second)
	fmt.Println(payload)
	return nil
}

func main() {
	cfg := httptask.Config{
		ServerAddr: "127.0.0.1:8081",
		TaskAddr:   "0.0.0.0:8082",
		Debug:      true,
	}
	server, err := httptask.NewServer(cfg)
	if err != nil {
		panic(err)
	}
	server.SetTask(&Hello{})

	err = server.Serve(context.Background())
	if err != nil {
		panic(err)
	}
}
