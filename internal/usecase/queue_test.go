package usecase

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"downloader/internal/domain"
)

type recorder struct {
	mu  sync.Mutex
	all []JobStatus
}

func (r *recorder) add(st JobStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.all = append(r.all, st)
}

// states returns the states one job went through, without repeats.
func (r *recorder) states(id int) []JobState {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []JobState
	for _, st := range r.all {
		if st.ID == id && (len(out) == 0 || out[len(out)-1] != st.State) {
			out = append(out, st.State)
		}
	}
	return out
}

func (r *recorder) last(id int) JobStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, st := range slices.Backward(r.all) {
		if st.ID == id {
			return st
		}
	}
	return JobStatus{}
}

func linkJob() domain.Job {
	return domain.Job{URL: "https://a.com", SavePath: "/tmp", Transcribe: true, Options: opts}
}

func TestQueue_RunsJobsInOrder(t *testing.T) {

	r := &recorder{}
	q := NewQueue(NewProcessUseCase(&FakeDownloader{}, &FakeTranscriber{}), r.add)

	id1, _ := q.Add(linkJob())
	id2, _ := q.Add(domain.Job{FilePath: "/a.mp3", Transcribe: true, Options: opts})
	q.workerDone.Wait()

	if got := r.states(id1); !slices.Equal(got, []JobState{Waiting, Downloading, Transcribing, Done}) {
		t.Errorf("job 1 states = %v", got)
	}
	if got := r.states(id2); !slices.Equal(got, []JobState{Waiting, Transcribing, Done}) {
		t.Errorf("job 2 states = %v", got)
	}
	if res := r.last(id1).Result; res.TextPath != "/tmp/video.mp3.txt" {
		t.Errorf("job 1 result = %+v", res)
	}
	if q.Busy() {
		t.Errorf("queue should be idle")
	}
}

func TestQueue_RejectsInvalidJob(t *testing.T) {

	q := NewQueue(NewProcessUseCase(&FakeDownloader{}, &FakeTranscriber{}), func(JobStatus) {})

	if _, err := q.Add(domain.Job{}); !errors.Is(err, ErrNoSource) {
		t.Errorf("Add() error = %v, want ErrNoSource", err)
	}
}

func TestQueue_CancelRunningAndWaiting(t *testing.T) {

	r := &recorder{}
	d := &FakeDownloader{Block: true}
	q := NewQueue(NewProcessUseCase(d, &FakeTranscriber{}), r.add)

	running, _ := q.Add(linkJob())
	waiting, _ := q.Add(linkJob())

	waitFor(t, func() bool { return r.last(running).State == Downloading })

	q.Cancel(waiting)
	q.Cancel(running)
	q.workerDone.Wait()

	if st := r.last(running).State; st != Canceled {
		t.Errorf("running job state = %v, want Canceled", st)
	}
	if st := r.last(waiting).State; st != Canceled {
		t.Errorf("waiting job state = %v, want Canceled", st)
	}
	if len(d.Calls) != 1 {
		t.Errorf("waiting job should never start, downloader calls = %d", len(d.Calls))
	}
}

func TestQueue_CloseStopsEverything(t *testing.T) {

	r := &recorder{}
	q := NewQueue(NewProcessUseCase(&FakeDownloader{Block: true}, &FakeTranscriber{}), r.add)

	id1, _ := q.Add(linkJob())
	id2, _ := q.Add(linkJob())
	waitFor(t, func() bool { return r.last(id1).State == Downloading })

	q.Close()

	if r.last(id1).State != Canceled || r.last(id2).State != Canceled {
		t.Errorf("states after Close = %v, %v", r.last(id1).State, r.last(id2).State)
	}
	if _, err := q.Add(linkJob()); !errors.Is(err, ErrQueueClosed) {
		t.Errorf("Add after Close error = %v", err)
	}
}

func TestQueue_FailedJobKeepsError(t *testing.T) {

	r := &recorder{}
	q := NewQueue(NewProcessUseCase(&FakeDownloader{Err: errors.New("bad link")}, &FakeTranscriber{}), r.add)

	id, _ := q.Add(linkJob())
	q.workerDone.Wait()

	st := r.last(id)
	if st.State != Failed || st.Err == nil || st.Err.Error() != "bad link" {
		t.Errorf("status = %+v", st)
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
