package events

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Event struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	ResourceID    string    `json:"resourceId,omitempty"`
	CorrelationID string    `json:"correlationId,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Payload       any       `json:"payload"`
}
type Broker struct {
	mu      sync.RWMutex
	next    int
	eventID uint64
	subs    map[int]chan Event
	history []Event
}

func New() *Broker { return &Broker{subs: map[int]chan Event{}} }
func (b *Broker) Subscribe() (int, <-chan Event, func()) {
	id, ch, unsubscribe, _ := b.SubscribeAfter("")
	return id, ch, unsubscribe
}

// SubscribeAfter replays the in-memory event window after Last-Event-ID. The
// caller must reconcile jobs through the REST endpoint when the process was
// restarted or the requested id falls outside this bounded window.
func (b *Broker) SubscribeAfter(lastID string) (int, <-chan Event, func(), []Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.next
	b.next++
	ch := make(chan Event, 32)
	b.subs[id] = ch
	replay := b.eventsAfter(lastID)
	return id, ch, func() {
		b.mu.Lock()
		if c, ok := b.subs[id]; ok {
			delete(b.subs, id)
			close(c)
		}
		b.mu.Unlock()
	}, replay
}
func (b *Broker) Publish(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	b.mu.Lock()
	b.eventID++
	e.ID = "evt-" + strconv.FormatUint(b.eventID, 10)
	b.history = append(b.history, e)
	if len(b.history) > 1024 {
		b.history = append([]Event(nil), b.history[len(b.history)-1024:]...)
	}
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
		}
	}
	b.mu.Unlock()
}
func (e Event) JSON() []byte { b, _ := json.Marshal(e); return b }

func (b *Broker) eventsAfter(lastID string) []Event {
	last := parseEventID(lastID)
	if last == 0 {
		return nil
	}
	result := make([]Event, 0)
	for _, event := range b.history {
		if parseEventID(event.ID) > last {
			result = append(result, event)
		}
	}
	return result
}

func parseEventID(value string) uint64 {
	parsed, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(value), "evt-"), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
