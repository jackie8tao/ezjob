package proto

import (
	"context"
)

type EventHandler func(ctx context.Context, evt *Event)

type Module interface {
	Boot(ctx context.Context) error
	Close() error
}
