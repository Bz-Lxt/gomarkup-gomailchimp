package provider

import (
	"sync"
	"testing"
)

func TestBalancerConcurrentPickAndReport(t *testing.T) {
	b := NewBalancer([]Health{
		{Key: "primary", Weight: 3, Score: 1, State: "closed"},
		{Key: "secondary", Weight: 1, Score: 1, State: "closed"},
	})

	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 24; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			for i := 0; i < 2000; i++ {
				h, ok := b.Pick()
				if !ok {
					t.Errorf("worker %d: no provider available", id)
					return
				}
				if i%100 == 0 {
					b.Report(h.Key, true)
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
}
