package metrics_test

import (
	"github.com/GeoNet/fdsn/internal/platform/metrics"
	"testing"
	"time"
)

func TestTimers(t *testing.T) {
	tm := metrics.ReadTimers()

	if len(tm) != 0 {
		t.Error("expected no timer stats")
	}

	for i := 0; i <= 5; i++ {
		tmr := metrics.Start()

		time.Sleep(time.Millisecond * 5)

		err := tmr.Track("test")
		if err != nil {
			t.Error(err)
		}
	}

	tm = metrics.ReadTimers()

	if len(tm) != 1 {
		t.Errorf("expected 1 timer stat got %d", len(tm))
	}
}
