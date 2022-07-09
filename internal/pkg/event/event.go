package event

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	pb "github.com/jackie8tao/ezjob/proto"
	log "github.com/sirupsen/logrus"
)

var (
	gManager *Manager
	lock     *sync.Mutex
)

type Manager struct {
	stopped  bool
	evtSub   message.Subscriber
	evtPub   message.Publisher
	handlers map[pb.EventType][]pb.EventHandler
	logger   log.FieldLogger
}

func init() {
	lock = &sync.Mutex{}
}

func NewManager() *Manager {
	lock.Lock()
	defer lock.Unlock()

	if gManager != nil {
		return gManager
	}

	chCfg := gochannel.Config{
		OutputChannelBuffer:            100,
		Persistent:                     false,
		BlockPublishUntilSubscriberAck: false,
	}
	pubSub := gochannel.NewGoChannel(chCfg, watermill.NopLogger{})

	gManager = &Manager{
		evtSub:   pubSub,
		evtPub:   pubSub,
		handlers: map[pb.EventType][]pb.EventHandler{},
		logger:   log.WithField("module", "event"),
		stopped:  false,
	}

	return gManager
}

func (m *Manager) Boot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	msgCh, err := m.evtSub.Subscribe(context.Background(), pb.EventKey)
	if err != nil {
		return err
	}
	go m.eventLoop(ctx, msgCh)

	m.logger.Debugf("event booted")

	return nil
}

func (m *Manager) Close() error {
	m.stopped = true

	err := m.evtPub.Close()
	if err != nil {
		return err
	}

	err = m.evtSub.Close()
	if err != nil {
		return err
	}

	m.logger.Debug("event exited")

	return nil
}

func (m *Manager) SubEvent(evtType pb.EventType, handlers []pb.EventHandler) {
	items, ok := m.handlers[evtType]
	if !ok {
		m.handlers[evtType] = handlers
		return
	}

	for _, v := range handlers {
		items = append(items, v)
	}
	m.handlers[evtType] = items
	return
}

func (m *Manager) PubEvent(evt *pb.Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	err = m.evtPub.Publish(pb.EventKey, message.NewMessage(watermill.NewUUID(), data))
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) eventLoop(ctx context.Context, msgCh <-chan *message.Message) {
	for {
		select {
		case msg := <-msgCh:
			if m.stopped {
				return
			}

			m.logger.Debugf("id: %s, payload: %s", msg.UUID, msg.Payload)

			evt := &pb.Event{}
			err := json.Unmarshal(msg.Payload, evt)
			if err != nil {
				panic(err)
			}
			go m.runHandlers(ctx, evt, m.handlers[evt.Type])
			msg.Ack()
		case <-ctx.Done():
			m.logger.Debugf("event loop existed")
			return
		}
	}
}

func (m *Manager) runHandlers(ctx context.Context, event *pb.Event, handlers []pb.EventHandler) {
	if len(handlers) <= 0 {
		return
	}

	innerCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(len(handlers))
	for _, f := range handlers {
		go func(handler pb.EventHandler) {
			defer wg.Done()
			handler(innerCtx, event)
		}(f)
	}
	wg.Wait()
}
