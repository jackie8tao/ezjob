package watcher

import (
	"context"
	"encoding/json"

	"github.com/jackie8tao/ezjob/pkg/event"
	pb "github.com/jackie8tao/ezjob/proto"
	log "github.com/sirupsen/logrus"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Watcher struct {
	stopped bool
	logger  log.FieldLogger
	cli     *clientv3.Client
	evtMgr  *event.Manager
}

func NewWatcher(cli *clientv3.Client, evtMgr *event.Manager) *Watcher {
	return &Watcher{
		logger:  log.WithField("module", "watcher"),
		cli:     cli,
		evtMgr:  evtMgr,
		stopped: false,
	}
}

func (w *Watcher) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	w.setupEventHandlers()

	w.logger.Debug("watcher booted")

	return nil
}

func (w *Watcher) Close() error {
	w.stopped = true

	err := w.cli.Close()
	if err != nil {
		return err
	}

	w.logger.Debug("watcher exited")

	return nil
}

func (w *Watcher) setupEventHandlers() {
	for k, v := range w.handlers() {
		w.evtMgr.SubEvent(k, []pb.EventHandler{v})
	}
}

func (w *Watcher) handlers() map[pb.EventType]pb.EventHandler {
	return map[pb.EventType]pb.EventHandler{
		pb.EventType_RoleChanged: w.roleChangedHandler,
	}
}

func (w *Watcher) roleChangedHandler(ctx context.Context, evt *pb.Event) {
	if evt.Type != pb.EventType_RoleChanged {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	switch evt.RoleChanged.NewRole {
	case pb.RoleLeader:
		w.stopped = false
		go w.watchTask()
		go w.watchTrigger()
	default:
		w.stopped = true
	}
}

func (w *Watcher) watchTask() {
	w.logger.Debug("watch tasks started")
	ch := w.cli.Watch(context.Background(), pb.JobKey, clientv3.WithPrefix())
	for {
		if w.stopped {
			w.logger.Debug("watch tasks exited")
			return
		}

		select {
		case rsp := <-ch:
			if rsp.Canceled {
				return
			}
			w.pubJobModifiedEvt()
		}
	}
}

func (w *Watcher) watchTrigger() {
	w.logger.Debug("watch trigger started")
	ch := w.cli.Watch(context.Background(), pb.TriggerKey, clientv3.WithPrefix())
	for {
		if w.stopped {
			w.logger.Debug("watch trigger exited")
			return
		}

		select {
		case rsp := <-ch:
			if rsp.Canceled {
				return
			}

			for _, v := range rsp.Events {
				if v.Type != mvccpb.PUT {
					continue
				}

				job := &pb.Job{}
				err := json.Unmarshal(v.Kv.Value, job)
				if err != nil {
					w.logger.Errorf("job error: %v", err)
					continue
				}
				w.pubJobDispatchedEvt(job)
			}
		}
	}
}

func (w *Watcher) pubJobModifiedEvt() {
	var err error
	defer func() {
		if err != nil {
			w.logger.Errorf("publish task_modified event error: %v", err)
		}
	}()

	evt := &pb.Event{
		Type:        pb.EventType_JobModified,
		JobModified: &pb.PayloadJobModified{},
	}

	err = w.evtMgr.PubEvent(evt)
	if err != nil {
		return
	}

	return
}

func (w *Watcher) pubJobDispatchedEvt(job *pb.Job) {
	var err error
	defer func() {
		if err != nil {
			w.logger.Errorf("publish task_dispatched event error: %v", err)
		}
	}()

	evt := &pb.Event{
		Type: pb.EventType_JobDispatched,
		JobDispatched: &pb.PayloadJobDispatched{
			Job: job,
		},
	}

	err = w.evtMgr.PubEvent(evt)
	if err != nil {
		return
	}

	return
}
