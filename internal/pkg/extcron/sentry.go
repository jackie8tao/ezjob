package extcron

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jackie8tao/ezjob/internal/model"
	"github.com/jackie8tao/ezjob/internal/pkg/event"
	pb "github.com/jackie8tao/ezjob/proto"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	etcdcli "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

type Sentry struct {
	engine *cron.Cron
	db     *gorm.DB
	cli    *etcdcli.Client
	logger logrus.FieldLogger
	evtMgr *event.Manager

	isRetrying bool
	retryLock  sync.Locker

	isAlarming bool
	alarmLock  sync.Locker
}

func NewSentry(cli *etcdcli.Client, db *gorm.DB, evtMgr *event.Manager) *Sentry {
	return &Sentry{
		engine:     cron.New(),
		db:         db,
		cli:        cli,
		logger:     logrus.WithField("module", "sentry"),
		evtMgr:     evtMgr,
		isRetrying: false,
		retryLock:  &sync.RWMutex{},
		isAlarming: false,
		alarmLock:  &sync.RWMutex{},
	}
}

func (s *Sentry) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	s.setupEventHandlers()

	err := s.setupJobs()
	if err != nil {
		return err
	}

	s.logger.Debug("sentry booted")

	return nil
}

func (s *Sentry) Close() error {
	err := s.cli.Close()
	if err != nil {
		return err
	}

	ctx := s.engine.Stop()
	select {
	case <-ctx.Done():
		s.logger.Debug("sentry exited")
	}

	return nil
}

func (s *Sentry) setupJobs() error {
	type taskDesc struct {
		Sched   string
		Handler cron.FuncJob
	}

	tasks := []taskDesc{
		{Sched: "*/2 * * * *", Handler: s.taskRetryHandler},
	}

	for _, v := range tasks {
		_, err := s.engine.AddFunc(v.Sched, v.Handler)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Sentry) setupEventHandlers() {
	for k, v := range s.handlers() {
		s.evtMgr.SubEvent(k, []pb.EventHandler{v})
	}
}

func (s *Sentry) handlers() map[pb.EventType]pb.EventHandler {
	return map[pb.EventType]pb.EventHandler{
		pb.EventType_RoleChanged: s.roleChangedHandler,
	}
}

func (s *Sentry) roleChangedHandler(ctx context.Context, evt *pb.Event) {
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
		s.engine.Start()
		s.logger.Debug("sentry started")
	case pb.RoleFollower, pb.RoleDeader:
		s.engine.Stop()
		s.logger.Debug("sentry stopped")
	default:
	}
}

func (s *Sentry) taskRetryHandler() {
	if s.isRetrying {
		s.logger.Debug("skip retry")
		return
	}

	var err error
	defer func() {
		if err != nil {
			s.logger.Errorf("retry job error: %v", err)
		}
	}()

	s.retryLock.Lock()
	defer s.retryLock.Unlock()

	s.logger.Debug("retry job started")

	s.isRetrying = true
	defer func() {
		s.isRetrying = false
	}()

	var execs []model.Execution
	err = s.db.Where("status = ?", pb.JobStatusFail).Where("retry_count < ?", 3).
		Find(&execs).
		Error
	if err != nil {
		return
	}

	var job *pb.Job

	for _, v := range execs {
		job, err = s.getJob(v.TaskName)
		if err != nil {
			s.logger.Warn(err)
			continue
		}
		err = s.updateExec(job, v)
		if err != nil {
			continue
		}
	}

	s.logger.Debug("retry job finished")

	return
}

func (s *Sentry) updateExec(job *pb.Job, exec model.Execution) error {
	db := s.db.Begin()
	defer db.Rollback()

	exec.RetryCount++
	err := db.Save(&exec).Error
	if err != nil {
		return err
	}

	err = s.pubTaskDispatchedEvt(job, int32(exec.ID))
	if err != nil {
		return err
	}

	db.Commit()

	return nil
}

func (s *Sentry) getJob(taskName string) (*pb.Job, error) {
	rsp, err := s.cli.Get(context.Background(), pb.GenJobKey(taskName))
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) <= 0 {
		return nil, fmt.Errorf("invalid job: %s", taskName)
	}

	job := &pb.Job{}
	err = json.Unmarshal(rsp.Kvs[0].Value, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *Sentry) pubTaskDispatchedEvt(job *pb.Job, execId int32) error {
	evt := &pb.Event{
		Type: pb.EventType_JobDispatched,
		JobDispatched: &pb.PayloadJobDispatched{
			Job:    job,
			ExecId: execId,
		},
	}

	err := s.evtMgr.PubEvent(evt)
	if err != nil {
		return err
	}

	return nil
}
