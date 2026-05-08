package clock

import (
	"testing"
	"time"
)

func TestSetSystemLocationToChina(t *testing.T) {
	original := time.Local
	t.Cleanup(func() {
		time.Local = original
	})

	if err := SetSystemLocationToChina(); err != nil {
		t.Fatalf("SetSystemLocationToChina() error = %v", err)
	}

	if got := time.Local.String(); got != ChinaTimezoneName {
		t.Fatalf("time.Local = %q, want %q", got, ChinaTimezoneName)
	}

	now := time.Now()
	if got := now.Location().String(); got != ChinaTimezoneName {
		t.Fatalf("time.Now() location = %q, want %q", got, ChinaTimezoneName)
	}

	_, offset := now.Zone()
	if offset != 8*60*60 {
		t.Fatalf("time.Now() offset = %d, want %d", offset, 8*60*60)
	}
}
