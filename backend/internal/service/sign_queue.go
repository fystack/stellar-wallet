package service

import (
	"sync"

	"go.uber.org/zap"
)

// Envelopes expire 300 s after they are built, so signing allows a little more concurrency than keygen.
const maxInflightSigns = 2

type signJob struct {
	walletID string
	network  string
	hash     []byte
}

type signQueue struct {
	dispatchQueue
	jobsMu sync.Mutex
	jobs   map[string]signJob
}

func (q *signQueue) add(transactionID string, job signJob) []string {
	q.jobsMu.Lock()
	if q.jobs == nil {
		q.jobs = map[string]signJob{}
	}
	q.jobs[transactionID] = job
	q.jobsMu.Unlock()
	return q.push(transactionID)
}

func (q *signQueue) take(transactionID string) (signJob, bool) {
	q.jobsMu.Lock()
	defer q.jobsMu.Unlock()
	job, ok := q.jobs[transactionID]
	delete(q.jobs, transactionID)
	return job, ok
}

// enqueueSign sends the transaction to the signer now when a slot is free; otherwise it waits in line.
// It returns the dispatch error only for the transaction being enqueued, so the caller can report it.
func (s *TransactionService) enqueueSign(transactionID string, job signJob) error {
	var dispatchErr error
	for _, id := range s.signs.add(transactionID, job) {
		if err := s.dispatchSign(id); err != nil && id == transactionID {
			dispatchErr = err
		}
	}
	return dispatchErr
}

func (s *TransactionService) signFinished(transactionID string) {
	for _, id := range s.signs.done(transactionID) {
		if err := s.dispatchSign(id); err != nil {
			zap.L().Warn("sign dispatch failed", zap.String("transaction", id), zap.Error(err))
		}
	}
}

func (s *TransactionService) dispatchSign(transactionID string) error {
	job, ok := s.signs.take(transactionID)
	if !ok {
		s.signFinished(transactionID)
		return nil
	}
	if err := s.mpc.Sign(job.walletID, transactionID, job.network, job.hash); err != nil {
		s.failSigning(transactionID, "signing could not be started")
		return err
	}
	s.WatchSignTimeout(transactionID)
	return nil
}
