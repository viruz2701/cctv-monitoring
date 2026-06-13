package worker

import (
	"context"
	"gb-telemetry-collector/internal/sip"
	"log/slog"
	"sync"
)

type Job struct {
	Message *sip.MessageContext
}

type WorkerPool struct {
	jobQueue   chan Job
	wg         sync.WaitGroup
	numWorkers int
	processor  sip.MessageProcessor
	logger     *slog.Logger
}

func NewWorkerPool(numWorkers int, processor sip.MessageProcessor, logger *slog.Logger) *WorkerPool {
	return &WorkerPool{
		jobQueue:   make(chan Job, 1000), // буферизованный канал
		numWorkers: numWorkers,
		processor:  processor,
		logger:     logger,
	}
}

func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
	p.logger.Info("Worker pool started", "workers", p.numWorkers)
}

func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			p.logger.Debug("Worker stopping", "id", id)
			return
		case job := <-p.jobQueue:
			p.processor.ProcessMessage(job.Message)
		}
	}
}

func (p *WorkerPool) Submit(job Job) {
	select {
	case p.jobQueue <- job:
		// ok
	default:
		p.logger.Warn("Job queue full, dropping message")
	}
}

func (p *WorkerPool) Stop() {
	close(p.jobQueue)
	p.wg.Wait()
}
