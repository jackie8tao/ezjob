package extcron

import (
	"context"
	"encoding/json"

	"github.com/jackie8tao/ezjob/internal/pkg/event"
	pb "github.com/jackie8tao/ezjob/proto"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

type Scheduler struct {
	stopped bool
	tasks   map[string]cron.EntryID
	engine  *cron.Cron
	cli     *clientv3.Client
	logger  logrus.FieldLogger
	evtMgr  *event.Manager
	db      *gorm.DB
}

func NewScheduler(cli *clientv3.Client, evtMgr *event.Manager, db *gorm.DB) *Scheduler {
	s := &Scheduler{
		stopped: false,
		engine:  cron.New(cron.WithSeconds()),
		tasks:   map[string]cron.EntryID{},
		cli:     cli,
		logger:  logrus.WithField("module", "extcron"),
		evtMgr:  evtMgr,
		db:      db,
	}

	return s
}

func (s *Scheduler) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	s.setupEventHandlers()

	s.logger.Debug("scheduler booted")

	return nil
}

func (s *Scheduler) Close() error {
	s.stop()

	err := s.cli.Close()
	if err != nil {
		return err
	}

	s.logger.Debug("scheduler exited")

	return nil
}

func (s *Scheduler) setupEventHandlers() {
	for k, v := range s.handlers() {
		s.evtMgr.SubEvent(k, []pb.EventHandler{v})
	}
}

func (s *Scheduler) handlers() map[pb.EventType]pb.EventHandler {
	return map[pb.EventType]pb.EventHandler{
		pb.EventType_RoleChanged: s.roleChangedHandler,
		pb.EventType_JobModified: s.jobModifiedHandler,
	}
}

func (s *Scheduler) roleChangedHandler(ctx context.Context, evt *pb.Event) {
	switch evt.RoleChanged.NewRole {
	case pb.RoleFollower:
		s.resetSchedTasks(ctx)
	case pb.RoleLeader:
		s.start()
	case pb.RoleDeader:
		s.stop()
	default:
	}
}

func (s *Scheduler) jobModifiedHandler(ctx context.Context, evt *pb.Event) {
	if evt.Type != pb.EventType_JobModified {
		return
	}

	s.stop()
	s.resetSchedTasks(ctx)
	s.start()
}

func (s *Scheduler) start() {
	s.engine.Start()
	s.stopped = false
	s.logger.Debug("scheduler started")
}

func (s *Scheduler) stop() {
	if s.stopped {
		return
	}

	stopCtx := s.engine.Stop()
	s.stopped = true
	select {
	case <-stopCtx.Done():
		s.logger.Debug("scheduler stopped")
		return
	}
}

func (s *Scheduler) resetSchedTasks(ctx context.Context) {
	var err error
	defer func() {
		if err != nil {
			s.logger.Errorf("reset scheduler tasks error: %v", err)
		}
	}()

	for _, v := range s.tasks {
		s.engine.Remove(v)
	}

	err = s.pullTasks(ctx)
	if err != nil {
		return
	}

	return
}

func (s *Scheduler) pullTasks(ctx context.Context) error {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend),
	}
	rsp, err := s.cli.Get(ctx, pb.JobKey, opts...)
	if err != nil {
		return err
	}

	jobs := make([]*pb.Job, 0)
	for _, v := range rsp.Kvs {
		job := &pb.Job{}
		err = json.Unmarshal(v.Value, job)
		if err != nil {
			return err
		}
		jobs = append(jobs, job)
	}

	var entryId cron.EntryID
	for _, v := range jobs {
		entryId, err = s.engine.AddFunc(v.Schedule, s.wrapTask(v))
		if err != nil {
			return err
		}
		s.tasks[v.Name] = entryId
	}

	return nil
}

func (s *Scheduler) wrapTask(job *pb.Job) cron.FuncJob {
	return func() {
		var err error
		defer func() {
			if err != nil {
				s.logger.Errorf("sched error: %v", err)
			}
		}()

		evt := &pb.Event{
			Type: pb.EventType_JobDispatched,
			JobDispatched: &pb.PayloadJobDispatched{
				Job: job,
			},
		}

		err = s.evtMgr.PubEvent(evt)
		if err != nil {
			return
		}
	}
}
