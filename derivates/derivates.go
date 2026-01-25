package derivates

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ImageFocusMode string

var (
	global     *Service
	globalOnce sync.Once
)

const (
	ImageFocusModeAuto   ImageFocusMode = "auto"
	ImageFocusModeManual ImageFocusMode = "manual"
	ImageFocusModeCenter ImageFocusMode = "center"
	ImageFocusModeTop    ImageFocusMode = "top"
	ImageFocusModeBottom ImageFocusMode = "bottom"
	ImageFocusModeLeft   ImageFocusMode = "left"
	ImageFocusModeRight  ImageFocusMode = "right"
)

type Key string

type ImageParams struct {
	FocusMode ImageFocusMode
	FocusX    *float32
	FocusY    *float32
	Rotation  int16
}

func (i *ImageParams) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("focus_mod", string(i.FocusMode)).
			Int16("rotation", i.Rotation)
		logging.Float32If(e, "focus_x", i.FocusX)
		logging.Float32If(e, "focus_y", i.FocusY)
	}
}

type Job struct {
	Key Key

	Image uint64
	Mode  config.DerivativeConfig

	SourcePath string
	TargetPath string

	ImageParams ImageParams
	Ctx         context.Context

	//	Done chan Result
}

func (j *Job) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("key", string(j.Key)).
			Uint64("image_id", j.Image).
			Str("derivate", j.Mode.Name).
			Str("path", j.TargetPath)
	}
}

/*
	type Result struct {
		Key      Key
		Duration time.Duration
		Err      error
	}
*/
type Step func(j *Job) error

type Service struct {
	mu      sync.Mutex
	cond    *sync.Cond
	queue   *list.List       // FIFO
	pending map[Key]struct{} // dedup:

	step    Step
	workers int

	closed bool
}

var (
	ErrClosed    = errors.New("derivate service is closed")
	ErrDuplicate = errors.New("job already queued/in-flight")
)

func NewService(step Step, workers int) *Service {
	if workers <= 0 {
		workers = 1
	}
	s := &Service{
		queue:   list.New(),
		pending: make(map[Key]struct{}, 1024),
		step:    step,
		workers: workers,
	}
	log.Logger.Info().Int("workers", workers).Msg("image derivate service created")
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *Service) Submit(j Job) (bool, error) {
	logg := logging.Enter(j.Ctx, "derivate.Service.Submit", map[string]any{"job": j})
	if j.Key == "" {
		err := fmt.Errorf("missing job key")
		logging.ExitErr(logg, err)
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		logging.ExitErr(logg, ErrClosed)
		return false, ErrClosed
	}
	if _, exists := s.pending[j.Key]; exists {
		logging.ExitErr(logg, ErrDuplicate)
		return false, ErrDuplicate
	}

	s.pending[j.Key] = struct{}{}
	s.queue.PushBack(&j)

	s.cond.Signal()
	logging.Exit(logg, "", nil)
	return true, nil
}

func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.cond.Broadcast()
}

func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(s.workers)

	for i := 0; i < s.workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			s.workerLoop(ctx, workerID)
		}(i + 1)
	}

	<-ctx.Done()
	s.cond.Broadcast()
	wg.Wait()
}

func (s *Service) workerLoop(ctx context.Context, workerID int) {
	logging.Enter(ctx, "service.derivate.workerloop", map[string]any{"worker Id": workerID})
	for {
		j := s.pop(ctx)
		if j == nil {
			logging.Error(nil, "service.derivate.loop", "error", "job is nil", nil)
			return
		}
		logg := logging.Enter(j.Ctx, "service.derivate.loop", map[string]any{"worker Id": workerID})

		res := "ok"
		err := s.execute(j)
		if err != nil {
			logging.ErrorContinue(logg, err, nil)
			res = "error"
		}

		s.mu.Lock()
		delete(s.pending, j.Key)
		s.mu.Unlock()

		/*
			if j.Done != nil {
				select {
				case j.Done <- Result{Key: j.Key, Duration: time.Since(start), Err: err}:
				default:
				}
			}
		*/
		logging.Exit(logg, res, nil)
	}
}

func (s *Service) pop(ctx context.Context) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if e := s.queue.Front(); e != nil {
			s.queue.Remove(e)
			return e.Value.(*Job)
		}

		if s.closed {
			return nil
		}
		if ctx.Err() != nil {
			return nil
		}

		s.cond.Wait()
	}
}

func (s *Service) execute(j *Job) error {
	return s.step(j)
}

func Init(ctx context.Context, workers int) {
	logging.Info("service.derivate", "init", "ok", "", nil)
	globalOnce.Do(func() {
		global = NewService(GenerateDerivateStep, workers)
		go global.Run(ctx)
	})
}

func Get() *Service {
	if global == nil {
		log.Logger.Panic().Str(logging.FieldFunc, "service.derivate.get").Str(logging.FieldEvent, "not initialized").Msg("")
		panic("derivate service not initialized")
	}
	return global
}

func Shutdown() {
	logging.Info("service.derivate", "shutdown", "ok", "", nil)
	if global != nil {
		global.Close()
	}
}
