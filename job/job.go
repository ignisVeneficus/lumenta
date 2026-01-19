package job

import "time"

type JobType string
type JobState string

const (
	JobIdle    JobState = "idle"    // nem fut
	JobRunning JobState = "running" // fut
	JobDone    JobState = "done"    // sikeresen befejezte
	JobFailed  JobState = "failed"  // hibával leállt
)

const (
	JobSync    JobType = "sync"
	JobRebuild JobType = "rebuild"
)

type JobStatus struct {
	Type     JobType
	State    JobState // idle | running | done | failed
	Current  int
	Total    int
	Message  string
	Started  time.Time
	Finished *time.Time
}

type JobManager interface {
	Start(job JobType) error
	Status(job JobType) JobStatus
}
