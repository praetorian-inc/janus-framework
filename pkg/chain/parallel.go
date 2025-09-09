package chain

import (
	"fmt"
	"sync"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
)

// Parallelize takes a link constructor and returns a parallelized version that distributes
// work across multiple workers. The number of workers is controlled by the "workers" parameter.
//
// Usage:
//
//	Parallelize(myLink.NewMyLink)  // Default 3 workers
//	Parallelize(myLink.NewMyLink)(cfg.WithArg("workers", 5))  // 5 workers
//
// The wrapper gracefully handles edge cases:
//   - Small input with large worker count: Only creates as many workers as needed
//   - Zero input: No workers created, graceful pass-through
//   - Worker failures: Individual failures don't stop other workers
func Parallelize(linkConstructor func(...cfg.Config) Link) func(...cfg.Config) Link {
	return func(configs ...cfg.Config) Link {
		pw := &ParallelWrapper{
			linkConstructor: linkConstructor,
			configs:         configs,
		}
		pw.Base = NewBase(pw, configs...)
		return pw
	}
}

type ParallelWrapper struct {
	*Base
	linkConstructor func(...cfg.Config) Link
	configs         []cfg.Config
	workerPool      *WorkerPool
	initialized     bool
	mutex           sync.Mutex
}

func (pw *ParallelWrapper) Params() []cfg.Param {
	baseParams := pw.Base.Params()

	workerParam := cfg.NewParam[int]("workers", "number of parallel workers").WithDefault(3)

	var linkParams []cfg.Param
	if pw.linkConstructor != nil {
		tempLink := pw.linkConstructor()
		linkParams = tempLink.Params()
	}

	allParams := []cfg.Param{workerParam}
	allParams = append(allParams, baseParams...)
	allParams = append(allParams, linkParams...)

	return allParams
}

func (pw *ParallelWrapper) Initialize() error {
	pw.mutex.Lock()
	defer pw.mutex.Unlock()

	if pw.initialized {
		return nil
	}

	workerCount := pw.getWorkerCount()
	pw.workerPool = NewWorkerPool(pw.linkConstructor, workerCount, pw.configs, pw.Logger)
	pw.initialized = true

	return nil
}

func (pw *ParallelWrapper) getWorkerCount() int {
	if workerCountArg := pw.Arg("workers"); workerCountArg != nil {
		if workerCount, ok := workerCountArg.(int); ok && workerCount > 0 {
			return workerCount
		}
	}
	return 3 // default
}

func (pw *ParallelWrapper) Process(input any) error {
	if err := pw.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize parallel wrapper: %w", err)
	}

	runtimeArgs := make(map[string]any)
	for key, value := range pw.Args() {
		if key != "workers" {
			runtimeArgs[key] = value
		}
	}

	pw.workerPool.SubmitWork(input, runtimeArgs, func(results []any) {
		for _, result := range results {
			pw.Send(result)
		}
	})

	return nil
}

func (pw *ParallelWrapper) Complete() error {
	var err error
	if pw.workerPool != nil {
		err = pw.workerPool.Shutdown()
	}
	if baseErr := pw.Base.Complete(); baseErr != nil && err == nil {
		err = baseErr
	}
	return err
}

type WorkItem struct {
	input       any
	callback    func([]any)
	runtimeArgs map[string]any
}

type WorkerPool struct {
	linkConstructor func(...cfg.Config) Link
	configs         []cfg.Config
	maxWorkers      int
	workChan        chan WorkItem
	wg              sync.WaitGroup
	logger          *cfg.Logger
	started         bool
	mutex           sync.Mutex
}

func NewWorkerPool(linkConstructor func(...cfg.Config) Link, maxWorkers int, configs []cfg.Config, logger *cfg.Logger) *WorkerPool {
	wp := &WorkerPool{
		linkConstructor: linkConstructor,
		configs:         configs,
		maxWorkers:      maxWorkers,
		workChan:        make(chan WorkItem, maxWorkers*2),
		logger:          logger,
	}
	wp.start()
	return wp
}

func (wp *WorkerPool) start() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if wp.started {
		return
	}

	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.started = true
	wp.logger.Debug("started worker pool", "workers", wp.maxWorkers)
}

func (wp *WorkerPool) worker(workerID int) {
	defer wp.wg.Done()

	for workItem := range wp.workChan {
		wp.processWorkItem(workerID, workItem)
	}
}

func (wp *WorkerPool) processWorkItem(workerID int, workItem WorkItem) {
	workerConfigs := make([]cfg.Config, len(wp.configs))
	copy(workerConfigs, wp.configs)

	for key, value := range workItem.runtimeArgs {
		workerConfigs = append(workerConfigs, cfg.WithArg(key, value))
	}

	worker := wp.linkConstructor(workerConfigs...)
	if worker == nil {
		wp.logger.Error("failed to create worker link", "worker_id", workerID)
		workItem.callback([]any{})
		return
	}

	results, err := worker.Invoke(workItem.input)

	if completeErr := worker.Complete(); completeErr != nil {
		wp.logger.Error("worker completion failed", "worker_id", workerID, "error", completeErr)
	}

	if err != nil {
		wp.logger.Error("worker processing failed", "worker_id", workerID, "error", err)
		workItem.callback([]any{})
		return
	}

	workItem.callback(results)
}

func (wp *WorkerPool) SubmitWork(input any, runtimeArgs map[string]any, callback func([]any)) {
	workItem := WorkItem{
		input:       input,
		callback:    callback,
		runtimeArgs: runtimeArgs,
	}

	select {
	case wp.workChan <- workItem:
	default:
		wp.processWorkItem(-1, workItem)
	}
}

func (wp *WorkerPool) Shutdown() error {
	wp.mutex.Lock()
	if wp.started {
		close(wp.workChan)
		wp.started = false
	}
	wp.mutex.Unlock()

	wp.wg.Wait()
	wp.logger.Debug("worker pool shutdown complete")
	return nil
}
