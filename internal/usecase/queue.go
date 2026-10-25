package usecase

import (
	"context"
	"errors"
	"slices"
	"sync"

	"downloader/internal/domain"
)

var ErrQueueClosed = errors.New("queue is closed")

type JobState int

const (
	Waiting JobState = iota
	Downloading
	Transcribing
	Done
	Failed
	Canceled
)

func (s JobState) Finished() bool { return s >= Done }

type JobStatus struct {
	ID      int
	Job     domain.Job
	State   JobState
	Percent float64 // 0-100, or Unknown
	Result  domain.Result
	Err     error
}

// Queue runs jobs one at a time, in the order they were added.
//
// onChange is called on every change, while the queue holds its lock,
// so calls arrive in order. It must be quick and must not call the Queue.
type Queue struct {
	process  *ProcessUseCase
	onChange func(JobStatus)

	mu         sync.Mutex
	ctx        context.Context
	stop       context.CancelFunc
	nextID     int
	pending    []JobStatus
	runningID  int // 0 = nothing running
	cancelJob  context.CancelFunc
	working    bool
	workerDone sync.WaitGroup
}

func NewQueue(p *ProcessUseCase, onChange func(JobStatus)) *Queue {
	ctx, stop := context.WithCancel(context.Background())
	return &Queue{process: p, onChange: onChange, ctx: ctx, stop: stop}
}

// Add puts a job at the end of the queue and returns its ID.
func (q *Queue) Add(job domain.Job) (int, error) {

	if err := Validate(job); err != nil {
		return 0, err
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.ctx.Err() != nil {
		return 0, ErrQueueClosed
	}

	q.nextID++
	st := JobStatus{ID: q.nextID, Job: job, State: Waiting, Percent: Unknown}
	q.pending = append(q.pending, st)
	q.onChange(st)

	if !q.working {
		q.working = true
		q.workerDone.Add(1)
		go q.work()
	}

	return st.ID, nil
}

// Cancel stops a running job or removes a waiting one.
func (q *Queue) Cancel(id int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if id == q.runningID {
		q.cancelJob()
		return
	}

	for i, st := range q.pending {
		if st.ID == id {
			q.pending = slices.Delete(q.pending, i, i+1)
			st.State = Canceled
			q.onChange(st)
			return
		}
	}
}

// CancelAll stops the running job and removes all waiting ones.
func (q *Queue) CancelAll() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cancelAllLocked()
}

func (q *Queue) cancelAllLocked() {
	if q.runningID != 0 {
		q.cancelJob()
	}
	for _, st := range q.pending {
		st.State = Canceled
		q.onChange(st)
	}
	q.pending = nil
}

// Busy reports whether a job is running or waiting.
func (q *Queue) Busy() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.runningID != 0 || len(q.pending) > 0
}

// Close cancels everything and waits until the running program has stopped.
func (q *Queue) Close() {
	q.mu.Lock()
	q.stop()
	q.cancelAllLocked()
	q.mu.Unlock()

	q.workerDone.Wait()
}

func (q *Queue) work() {
	defer q.workerDone.Done()

	for {
		q.mu.Lock()
		if len(q.pending) == 0 {
			q.working = false
			q.mu.Unlock()
			return
		}
		st := q.pending[0]
		q.pending = q.pending[1:]

		ctx, cancel := context.WithCancel(q.ctx)
		q.runningID = st.ID
		q.cancelJob = cancel
		q.mu.Unlock()

		res, err := q.process.Run(ctx, st.Job, func(stage Stage, p float64) {
			q.mu.Lock()
			defer q.mu.Unlock()

			st.State = Downloading
			if stage == StageTranscribe {
				st.State = Transcribing
			}
			st.Percent = p
			q.onChange(st)
		})
		canceled := ctx.Err() != nil
		cancel()

		q.mu.Lock()
		st.Result = res
		switch {
		case canceled:
			st.State = Canceled
		case err != nil:
			st.State = Failed
			st.Err = err
		default:
			st.State = Done
			st.Percent = 100
		}
		q.runningID = 0
		q.cancelJob = nil
		q.onChange(st)
		q.mu.Unlock()
	}
}
