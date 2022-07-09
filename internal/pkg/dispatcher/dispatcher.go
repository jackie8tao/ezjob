package dispatcher

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-http/v2/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackie8tao/ezjob/internal/model"
	"github.com/jackie8tao/ezjob/internal/pkg/event"
	pb "github.com/jackie8tao/ezjob/proto"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

type Dispatcher struct {
	cfg     *pb.DispatcherConfig
	cli     *clientv3.Client
	logger  logrus.FieldLogger
	trigger message.Publisher
	evtMgr  *event.Manager
	db      *gorm.DB
}

func NewDispatcher(cfg *pb.DispatcherConfig, cli *clientv3.Client, evtMgr *event.Manager, db *gorm.DB) *Dispatcher {
	d := &Dispatcher{
		cfg:    cfg,
		cli:    cli,
		logger: logrus.WithField("module", "dispatcher"),
		evtMgr: evtMgr,
		db:     db,
	}

	return d
}

func (d *Dispatcher) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	err := d.setupTrigger()
	if err != nil {
		return err
	}
	d.setupEventHandlers()

	d.logger.Debug("dispatcher booted")

	return nil
}

func (d *Dispatcher) Close() error {
	err := d.cli.Close()
	if err != nil {
		return err
	}

	err = d.trigger.Close()
	if err != nil {
		return err
	}

	d.logger.Debug("dispatcher exited")

	return nil
}

func (d *Dispatcher) setupTrigger() error {
	switch d.cfg.Type {
	case pb.DispatchTypeHttp:
		return d.initHttpTrigger()
	case pb.DispatchTypeKafka:
		return d.initKafkaTrigger()
	default:
		return d.initHttpTrigger()
	}
}

func (d *Dispatcher) initKafkaTrigger() error {
	return nil
}

func (d *Dispatcher) initHttpTrigger() error {
	cfg := http.PublisherConfig{
		MarshalMessageFunc: http.DefaultMarshalMessageFunc,
		Client: &stdhttp.Client{
			Timeout: 3 * time.Second,
		},
		DoNotLogResponseBodyOnServerError: false,
	}
	pub, err := http.NewPublisher(cfg, watermill.NewStdLogger(false, false))
	if err != nil {
		return err
	}

	d.trigger = pub
	return nil
}

func (d *Dispatcher) setupEventHandlers() {
	for k, v := range d.handlers() {
		d.evtMgr.SubEvent(k, []pb.EventHandler{v})
	}
}

func (d *Dispatcher) handlers() map[pb.EventType]pb.EventHandler {
	return map[pb.EventType]pb.EventHandler{
		pb.EventType_JobDispatched: d.jobDispatchedHandler,
	}
}

func (d *Dispatcher) jobDispatchedHandler(ctx context.Context, evt *pb.Event) {
	if evt.Type != pb.EventType_JobDispatched {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	job := evt.JobDispatched.Job
	if job.Proc == nil {
		return
	}

	var err error
	defer func() {
		if err != nil {
			d.logger.Errorf("job dispatched error: %v", err)
		}
	}()

	switch job.Mode {
	case pb.TaskModeSingle:
		var executions []model.Execution
		err = d.db.Where("status = ?", pb.TaskStatusDoing).Find(&executions).Error
		if err != nil {
			return
		}
		if len(executions) > 0 {
			d.logger.Debugf("skip job %s", job.Name)
			return
		}
	case pb.TaskModeMulti:
	}

	db := d.db.Begin()
	defer db.Callback()

	proc := job.Proc

	execId := evt.JobDispatched.ExecId
	if execId <= 0 {
		exec := model.Execution{
			TaskName:   job.Name,
			TaskOwners: strings.Join(job.Owners, ";"),
			Status:     pb.TaskStatusDoing,
			RetryCount: 0,
		}

		switch proc.Type {
		case pb.ProcType_HTTP:
			exec.Payload = proc.HttpProc.Payload
		case pb.ProcType_Kafka:
		default:
		}

		err = db.Create(&exec).Error
		if err != nil {
			return
		}

		execId = int32(exec.ID)
	}

	switch proc.Type {
	case pb.ProcType_HTTP:
		err = d.doHttpProc(job.Name, execId, proc.HttpProc)
	default:
	}
	if err != nil {
		return
	}

	db.Commit()

	d.logger.Debugf("job dispatched: %s", job.Name)

	return
}

func (d *Dispatcher) doHttpProc(jobName string, execId int32, proc *pb.HttpProc) error {
	payload := &pb.HttpProcPayload{
		TaskName:    jobName,
		ExecutionId: execId,
		Payload:     proc.Payload,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := message.NewMessage(watermill.NewUUID(), data)
	err = d.trigger.Publish(proc.Url, msg)
	if err != nil {
		return err
	}

	return nil
}
