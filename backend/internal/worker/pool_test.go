// Package worker — unit tests for WorkerPool.
// Соответствует:
//   - IEC 62443-3-3 SR 7.1 (Resource availability — worker pool sizing)
//   - ISO 27001 A.12.1.2 (Change management — graceful shutdown)
package worker

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"gb-telemetry-collector/internal/sip"
)

// mockProcessor реализует sip.MessageProcessor для тестирования.
type mockProcessor struct {
	mu       sync.Mutex
	messages []*sip.MessageContext
	delay    time.Duration
}

func (m *mockProcessor) ProcessMessage(msg *sip.MessageContext) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *mockProcessor) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func TestWorkerPool_StartAndStop(t *testing.T) {
	processor := &mockProcessor{}
	logger := slog.Default()
	pool := NewWorkerPool(5, processor, logger)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Отправляем несколько заданий
	for i := 0; i < 10; i++ {
		pool.Submit(Job{Message: &sip.MessageContext{}})
	}

	// Даём время на обработку
	time.Sleep(100 * time.Millisecond)

	cancel()
	pool.Stop()

	if processor.Count() != 10 {
		t.Errorf("expected 10 processed messages, got %d", processor.Count())
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	processor := &mockProcessor{delay: 50 * time.Millisecond}
	logger := slog.Default()
	pool := NewWorkerPool(1, processor, logger)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Заполняем очередь (буфер 1000)
	for i := 0; i < 1100; i++ {
		pool.Submit(Job{Message: &sip.MessageContext{}})
	}

	time.Sleep(200 * time.Millisecond)
	cancel()
	pool.Stop()

	// Должны быть обработаны не более 1000 (размер буфера)
	count := processor.Count()
	if count > 1000 {
		t.Errorf("expected <= 1000 processed (buffer size), got %d", count)
	}
}

func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	processor := &mockProcessor{}
	logger := slog.Default()
	pool := NewWorkerPool(10, processor, logger)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				pool.Submit(Job{Message: &sip.MessageContext{}})
			}
		}()
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)
	cancel()
	pool.Stop()

	if processor.Count() != 500 {
		t.Errorf("expected 500 processed messages, got %d", processor.Count())
	}
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	processor := &mockProcessor{delay: 10 * time.Millisecond}
	logger := slog.Default()
	pool := NewWorkerPool(4, processor, logger)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Отправляем задания
	for i := 0; i < 20; i++ {
		pool.Submit(Job{Message: &sip.MessageContext{}})
	}

	// Ждём часть обработки
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Stop должен дождаться завершения активных workers
	done := make(chan struct{})
	go func() {
		pool.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK — graceful shutdown завершён
	case <-time.After(2 * time.Second):
		t.Fatal("graceful shutdown timed out")
	}
}
