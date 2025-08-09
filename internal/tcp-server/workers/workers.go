package workers

import (
	"errors"
	"fmt"
	_type "github.com/Tagakama/ServerManager/internal/tcp-server/type"
	"sync"
)

type TaskSubmitter interface {
	AddTask(task _type.PendingConnection)
}

type Task struct {
	ID      int
	Request _type.PendingConnection
}

type WorkerPool struct {
	isClosed bool
	tasks    chan Task
	results  chan Result
	mu       sync.Mutex
}

type Result struct {
	TaskID int
	Output string
	Err    error
}

var TaskCount int = 1

func NewWorkerPool(numWorkers int) *WorkerPool {
	if numWorkers <= 0 {
		//return nil, errors.New("Invalid number of workers")
	}

	pool := &WorkerPool{
		isClosed: false,
		tasks:    make(chan Task, 100),
		results:  make(chan Result, 100),
		mu:       sync.Mutex{},
	}

	go pool.Proccess(numWorkers)
	fmt.Printf("Worker pool created with %d workers.\n", numWorkers)
	return pool
}

func (wp *WorkerPool) Proccess(numWorkers int) {
	wg := sync.WaitGroup{}
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for task := range wp.tasks {
				wp.results <- Result{
					TaskID: task.ID,
					Output: fmt.Sprintf("handled %d", task.ID),
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range wp.results {
			if result.Err != nil {
				fmt.Printf("Task results %d returned an error: %s\n", result.TaskID, result.Err)
			}
			//TODO Возврат ответа клиенту о статусе запроса
		}
	}()

	wg.Wait()
	//close(wp.results)
}

func (wp *WorkerPool) AddTask(task _type.PendingConnection) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if wp.isClosed {
		errors.New("Worker pool is closed")
	}

	select {
	case wp.tasks <- Task{TaskCount, task}:

	default:
		errors.New("Worker pool is full")
	}
}

func (p *WorkerPool) Submit(task Task) {
	p.tasks <- task
}

func (p *WorkerPool) GetResults() <-chan Result {
	return p.results
}

func (p *WorkerPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClosed {
		return errors.New("Worker pool is closed")
	}
	p.isClosed = true
	close(p.tasks)
	return nil
}
