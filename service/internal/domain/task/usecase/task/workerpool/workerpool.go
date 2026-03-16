package workerpool

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

type Task func(ctx context.Context, workerNo int) error

type Config struct {
	Workers       int
	RecoverPanics bool
}

type Pool struct {
	cfg Config

	ctx    context.Context
	cancel context.CancelFunc

	tasks chan request

	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once

	mu     sync.RWMutex
	closed bool
}

type request struct {
	task Task
	done chan error
}

func New(parent context.Context, cfg Config) (*Pool, error) {
	if cfg.Workers <= 0 {
		return nil, fmt.Errorf("Workers must be > 0")
	}
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	return &Pool{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
		tasks:  make(chan request),
	}, nil
}

func (p *Pool) Start() {
	p.startOnce.Do(func() {
		for workerNo := 1; workerNo <= p.cfg.Workers; workerNo++ {
			p.wg.Add(1)
			go p.workerLoop(workerNo)
		}
	})
}

func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		p.cancel()

		p.mu.Lock()
		if !p.closed {
			p.closed = true
			close(p.tasks)
		}
		p.mu.Unlock()

		p.wg.Wait()
	})
}

func (p *Pool) Submit(ctx context.Context, task Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-p.ctx.Done():
		return fmt.Errorf("pool stopped: %w", p.ctx.Err())
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	req := request{
		task: task,
		done: make(chan error, 1),
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("pool stopped: tasks channel is closed")
	}

	select {
	case <-p.ctx.Done():
		p.mu.RUnlock()
		return fmt.Errorf("pool stopped: %w", p.ctx.Err())
	case <-ctx.Done():
		p.mu.RUnlock()
		return ctx.Err()
	case p.tasks <- req:
		p.mu.RUnlock()
	}

	select {
	case <-p.ctx.Done():
		return fmt.Errorf("pool stopped while waiting result: %w", p.ctx.Err())
	case <-ctx.Done():
		return ctx.Err()
	case err := <-req.done:
		return err
	}
}

func (p *Pool) workerLoop(workerNo int) {
	defer p.wg.Done()

	for req := range p.tasks {
		var err error
		if p.cfg.RecoverPanics {
			err = p.runTaskRecover(req.task, workerNo)
		} else {
			err = req.task(p.ctx, workerNo)
		}
		select {
		case req.done <- err:
		default:
		}
	}
}

func (p *Pool) runTaskRecover(task Task, workerNo int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("task panic (worker %d): %v\n%s", workerNo, r, debug.Stack())
		}
	}()
	return task(p.ctx, workerNo)
}
