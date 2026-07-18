package api

import (
	"context"
	"log"
	"sync"
)

//go:generate go run ../cmd/genrealtime -contract ../../contracts/realtime.json -go realtime_gen.go -ts ../../web/src/realtime.gen.ts

// Event is published by MCP tool handlers and broadcast to WebSocket clients.
type Event struct {
	Type     EventType `json:"type"`
	Sequence uint64    `json:"sequence"`
	Payload  any       `json:"payload"`
}

type busSubscriber struct {
	ch       chan Event
	wake     chan struct{}
	lostFrom uint64
	lostTo   uint64
}

// Bus is a fan-out pub/sub for Events. Publishers call Publish; the WebSocket
// hub calls Subscribe to receive a channel of all events.
type Bus struct {
	mu          sync.Mutex
	subscribers []*busSubscriber
	sequence    uint64
}

func NewBus() *Bus { return &Bus{} }

// SubscribeContext returns a buffered channel that receives all future events.
func (b *Bus) SubscribeContext(ctx context.Context) chan Event {
	subscriber := &busSubscriber{ch: make(chan Event, 64), wake: make(chan struct{}, 1)}
	b.mu.Lock()
	b.subscribers = append(b.subscribers, subscriber)
	b.mu.Unlock()
	go b.runSubscriber(ctx, subscriber)
	return subscriber.ch
}

// Publish sends an event to all subscribers. Non-blocking: full channels are skipped.
func (b *Bus) Publish(e Event) {
	payload, err := DecodeRealtimePayload(e.Type, e.Payload)
	if err != nil {
		log.Printf("realtime contract rejected event type=%s: %v", e.Type, err)
		return
	}
	e.Payload = payload
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sequence++
	e.Sequence = b.sequence
	for _, subscriber := range b.subscribers {
		if subscriber.lostFrom != 0 {
			subscriber.lostTo = e.Sequence
			select {
			case subscriber.wake <- struct{}{}:
			default:
			}
			continue
		}
		select {
		case subscriber.ch <- e:
		default:
			if subscriber.lostFrom == 0 {
				subscriber.lostFrom = e.Sequence
			}
			subscriber.lostTo = e.Sequence
			select {
			case subscriber.wake <- struct{}{}:
			default:
			}
		}
	}
}

func (b *Bus) runSubscriber(ctx context.Context, subscriber *busSubscriber) {
	defer func() {
		b.mu.Lock()
		for i, current := range b.subscribers {
			if current == subscriber {
				b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
		close(subscriber.ch)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-subscriber.wake:
			if !b.deliverPendingResync(ctx, subscriber) {
				return
			}
		}
	}
}

func (b *Bus) deliverPendingResync(ctx context.Context, subscriber *busSubscriber) bool {
	for {
		b.mu.Lock()
		from, to := subscriber.lostFrom, subscriber.lostTo
		b.mu.Unlock()
		if from == 0 {
			return true
		}
		event := Event{
			Type:     EventResyncRequired,
			Sequence: to,
			Payload:  ResyncRequiredPayload{FromSequence: int64(from), ToSequence: int64(to)},
		}
		select {
		case subscriber.ch <- event:
		case <-ctx.Done():
			return false
		}

		b.mu.Lock()
		if subscriber.lostTo <= to {
			subscriber.lostFrom = 0
			subscriber.lostTo = 0
			b.mu.Unlock()
			return true
		}
		subscriber.lostFrom = to + 1
		b.mu.Unlock()
	}
}
