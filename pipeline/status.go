package pipeline

import (
	"sync"
	"sync/atomic"
)

type SyncStatus string

const (
	SyncStatusIdle    SyncStatus = "idle"
	SyncStatusRunning SyncStatus = "running"
)

var (
	runtime     *SyncRuntime
	runtimeOnce sync.Once
)

type SyncRuntime struct {
	syncID atomic.Uint64
	status atomic.Value // SyncStatus
}

func NewSyncRuntime() *SyncRuntime {
	s := &SyncRuntime{}
	s.status.Store(SyncStatusIdle)
	return s
}
func (s *SyncRuntime) Start(syncID uint64) bool {
	if s.status.Load().(SyncStatus) == SyncStatusRunning {
		return false
	}

	s.syncID.Store(syncID)
	s.status.Store(SyncStatusRunning)
	return true
}
func (s *SyncRuntime) Stop(syncID uint64) {
	current := s.syncID.Load()

	if current != syncID {
		return
	}
	s.status.Store(SyncStatusIdle)
	s.syncID.Store(0)
}
func (s *SyncRuntime) Get() (uint64, SyncStatus) {
	return s.syncID.Load(), s.status.Load().(SyncStatus)
}

func Global() *SyncRuntime {
	runtimeOnce.Do(func() {
		runtime = NewSyncRuntime()
	})
	return runtime
}
