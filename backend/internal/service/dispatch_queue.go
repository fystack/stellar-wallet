package service

import "sync"

// dispatchQueue limits how many MPC requests are in flight so a burst does not flood mpcium.
type dispatchQueue struct {
	limit    int
	mu       sync.Mutex
	pending  []string
	inflight map[string]bool
}

// push queues id and returns the ids that may be dispatched now.
func (q *dispatchQueue) push(id string) (next []string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, id)
	return q.drain()
}

// done frees id's slot and returns the ids that may be dispatched now.
func (q *dispatchQueue) done(id string) (next []string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.inflight[id] {
		return nil
	}
	delete(q.inflight, id)
	return q.drain()
}

func (q *dispatchQueue) drain() (next []string) {
	if q.inflight == nil {
		q.inflight = map[string]bool{}
	}
	limit := q.limit
	if limit < 1 {
		limit = 1
	}
	for len(q.pending) > 0 && len(q.inflight) < limit {
		id := q.pending[0]
		q.pending = q.pending[1:]
		q.inflight[id] = true
		next = append(next, id)
	}
	return next
}
