package producers

import (
	"testing"

	"github.com/guilhermelinosp/fast-listeners/internal/listeners"
)

func TestCorrelationHeaders(t *testing.T) {
	got := correlationHeaders(listeners.Event{ID: "e1", AggregateID: "o1", EventType: "order.requested"})
	want := map[string]string{"event_id": "e1", "order_id": "o1", "event_type": "order.requested"}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("header %s = %q, want %q", k, got[k], v)
		}
	}
}
