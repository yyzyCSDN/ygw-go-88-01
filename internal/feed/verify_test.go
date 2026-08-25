package feed_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"chemicalprocessdcs/internal/feed"
	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
)

func TestFeedRegisterConcurrent(t *testing.T) {
	params := param.NewStore()
	params.SetParams("main", model.ProcessParams{FeedTarget: 10})
	confirm := func(ctx context.Context, id string) (model.FeedConfirmation, error) {
		return model.FeedConfirmation{Amount: 1, OK: true}, nil
	}
	s := feed.NewService(params, confirm)

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Register(fmt.Sprintf("R%d", i))
		}(i)
	}
	wg.Wait()
	for i := 0; i < 200; i++ {
		if s.State(fmt.Sprintf("R%d", i)) == "" {
			t.Fatalf("reactor R%d not registered", i)
		}
	}
	if s.RegisteredCount() != 200 {
		t.Fatalf("registered count = %d, want 200", s.RegisteredCount())
	}
}
