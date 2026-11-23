package storage_test

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/storage"
	crand "crypto/rand"
	"math/rand/v2"
	"testing"
	"time"
)

func randomEvent() domain.Event {
	return domain.Event{
		Time:       time.UnixMilli(rand.Int64()),
		Level:      crand.Text(),
		Component:  crand.Text(),
		JobID:      rand.Int(),
		WorkerID:   rand.Int(),
		RangeID:    rand.Int(),
		ResourceID: rand.Int(),
		Operation:  crand.Text(),
		Result:     crand.Text(),
		ErrorCode:  rand.Int(),
		Message:    crand.Text(),
	}
}

func TestEvents(t *testing.T) {
	t.Run("Add and get an event", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		event := randomEvent()

		err = s.AddEvent(ctx, event)
		if err != nil {
			t.Fatalf("Failed to add an event: %v", err)
		}

		{
			stored, err := s.GetEvents(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || stored[0] != event {
				t.Fatalf("Received event isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetEventsByJobID(ctx, event.JobID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || stored[0] != event {
				t.Fatalf("Received event isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetEventsByWorkerID(ctx, event.WorkerID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || stored[0] != event {
				t.Fatalf("Received event isn't equal to the saved one, got %v", stored)
			}
		}
	})
}
