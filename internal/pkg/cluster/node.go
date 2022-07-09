package cluster

import (
	"context"
	"fmt"
	"os"

	"github.com/jackie8tao/ezjob/internal/pkg/event"
	pb "github.com/jackie8tao/ezjob/proto"
	log "github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type Node struct {
	hostname string
	role     string
	limit    int
	stopped  bool

	cli    *clientv3.Client
	logger log.FieldLogger
	sess   *concurrency.Session
	evtMgr *event.Manager
}

func NewNode(cli *clientv3.Client, evtMgr *event.Manager) *Node {
	hostname, err := os.Hostname()
	if err != nil {
		panic(fmt.Errorf("hostname error: %v", err))
	}

	obj := &Node{
		hostname: hostname,
		cli:      cli,
		evtMgr:   evtMgr,
		logger:   log.WithField("module", "cluster"),
		stopped:  false,
	}

	return obj
}

func (n *Node) setupSess() error {
	sess, err := concurrency.NewSession(n.cli)
	if err != nil {
		return err
	}

	n.sess = sess
	return nil
}

func (n *Node) closeSess() error {
	if n.sess == nil {
		return nil
	}

	err := n.sess.Close()
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) resetSess() error {
	err := n.closeSess()
	if err != nil {
		return err
	}

	err = n.setupSess()
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) clearLimit() {
	n.limit = 0
}

func (n *Node) setupEventHandlers() {
	for k, v := range n.handlers() {
		n.evtMgr.SubEvent(k, []pb.EventHandler{v})
	}
}

func (n *Node) handlers() map[pb.EventType]pb.EventHandler {
	return map[pb.EventType]pb.EventHandler{
		pb.EventType_RoleChanged: n.roleChangedHandler,
	}
}

func (n *Node) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	err := n.setupSess()
	if err != nil {
		panic(fmt.Errorf("setup etcd session error: %v", err))
	}

	n.setupEventHandlers()

	n.logger.Debugf("cluster node booted")

	return nil
}

func (n *Node) Close() error {
	n.stopped = true

	err := n.sess.Close()
	if err != nil {
		return err
	}

	err = n.cli.Close()
	if err != nil {
		return err
	}

	n.logger.Debugf("cluster node exited")

	return nil
}

func (n *Node) StartElection() {
	n.changeRole(pb.RoleFollower)
}

func (n *Node) roleChangedHandler(ctx context.Context, evt *pb.Event) {
	if evt.Type != pb.EventType_RoleChanged {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	switch evt.RoleChanged.NewRole {
	case pb.RoleFollower:
		go n.doElect()
	case pb.RoleLeader:
		n.clearLimit()
		go n.doWatch()
	case pb.RoleDeader:
	default:
	}
}

func (n *Node) doElect() {
	var err error
	defer func() {
		if err != nil {
			n.logger.Errorf("elect error: %v", err)
		}
	}()

elect:
	n.logger.Debugf("do elect at %d", n.limit)
	if n.limit > 3 {
		n.changeRole(pb.RoleDeader)
		return
	}

	n.limit++
	e := concurrency.NewElection(n.sess, pb.ElectionKey)
	err = e.Campaign(context.Background(), n.hostname)
	if err != nil {
		goto elect
	}

	n.changeRole(pb.RoleLeader)
	return
}

func (n *Node) doWatch() {
	var err error
	defer func() {
		if err != nil {
			n.logger.Errorf("watch error: %v", err)
		}
	}()

	select {
	case <-n.sess.Done():
		if n.stopped {
			return
		}

		err = n.resetSess()
		if err != nil {
			n.changeRole(pb.RoleDeader)
		} else {
			n.changeRole(pb.RoleFollower)
		}
	}
}

func (n *Node) changeRole(role string) {
	n.logger.Debugf("role changed: %s -> %s", n.role, role)

	oldRole := n.role
	n.role = role

	err := n.publishRoleChangedMsg(oldRole, role)
	if err != nil {
		panic(err)
	}
}

func (n *Node) publishRoleChangedMsg(oldRole, newRole string) error {
	evt := &pb.Event{
		Type: pb.EventType_RoleChanged,
		RoleChanged: &pb.PayloadRoleChanged{
			OldRole: oldRole,
			NewRole: newRole,
		},
	}

	err := n.evtMgr.PubEvent(evt)
	if err != nil {
		return err
	}

	return nil
}
