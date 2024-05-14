package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// This file contains code to allow S3 objects to be downloaded from S3 concurrently,
// while still retaining the order that the objects were given in.
// The pattern has been adapted from https://www.wwt.com/article/fan-out-fan-in-while-maintaining-order

// General flow:

// 0. To ensure memory usage at any one time doesn't get too big, enough memory to contain the size of all
//    the S3 Objects requested is secured from the memory pool before starting the work.
// 1. When the Process function is called, it creates a worker group and takes the specified max number of
//    workers per request from the worker pool.
// 2. The Process function returns the output channel which will have the desired HydratedFiles in order.
//    This needs to be read from to ensure this whole process is not blocked.
// 3. Each worker adds themselves to the the WorkerGroup's roster, indicating they are ready to receive work.
// 4. When a job comes in, the WorkerGroup's AddWork function gets the first available worker from
//    the roster channel and adds that worker's output channel to its reception channel in the order that workers
//    are added to the roster. Since work is given to workers in the order that they are in the roster, this
//    means it doesn't matter how long each piece of work takes, because all the workers' output is read from
//    reception in order, which will block if needed to ensure the WorkerGroup's final output is in the same order
//    as it came in.
// 5. The Process function receives work from the jobs channel. Each job is an S3 Object to download.
// 6. FileProcessor defines a function that turns a job into a HydratedFile. This is supplied to the Process function.
// 7. After a worker finishes a piece of work, it adds itself back to the roster, and releases the memory for that
//    object back to the pool.
// 8. The WorkerGroup's startOuput function acts a bridge, where it ranges over the reception channel, gets each
//    worker output channel, takes the HydratedFile from it, and sends it to its output channel.
// 9. When the jobs channel is closed, and the last piece of work has been added to the WorkerGroup,
//    the WorkerGroup calls its stopWork function. This utilises Go's context package to cancel the process. This will
//    unblock the WorkerGroup's cleanup function which waits for the cancel to happen before waiting for all workers
//    to be finished via the WorkerGroup's sync.WaitGroup, after which it closes up the remaining channels.
// 10. Each worker is returned to the worker pool.

type S3Concurrent struct {
	S3
	manager *ConcurrencyManager
}

type ConcurrencyManager struct {
	memoryPool           memoryPool
	workerPool           workerPool
	memoryTotalSize      int64
	memoryChunkSize      int64
	maxWorkersPerRequest int
}

type memoryPool struct {
	channel chan int64
	mutex   sync.Mutex
}

type workerPool struct {
	channel chan *worker
	mutex   sync.Mutex
}

type FileProcessor func(types.Object) HydratedFile

type HydratedFile struct {
	Key   string
	Data  []byte
	Error error
}

type worker struct {
	manager *ConcurrencyManager
	input   chan types.Object
	output  chan HydratedFile
}

// NewConcurrent returns an S3Concurrent client, which embeds an S3 client, and has a ConcurrencyManager
// to allow the use of the GetAllConcurrently function. The GetAllConcurrently function can download multiple files
// at once while retaining order. The S3 client is configured to make use of the specified maxConnections.
// Also, ensure that the S3 Client has access to maxBytes in memory to avoid out of memory errors.
func NewConcurrent(maxConnections, maxConnectionsPerRequest, maxBytes int) (S3Concurrent, error) {

	// Create S3 client with custom HTTP client to facilitate higher concurrency.
	var err error
	htmlClientOption := func(options *s3.Options) {
		httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(t *http.Transport) {
			t.MaxIdleConns = maxConnections
			t.MaxIdleConnsPerHost = maxConnections
		})
		options.HTTPClient = httpClient
	}
	s3Client, err := NewWithOptions(htmlClientOption)
	if err != nil {
		return S3Concurrent{}, fmt.Errorf("error creating base S3 Client for S3Concurrent: %w", err)
	}
	if maxConnections <= 0 || maxConnectionsPerRequest <= 0 || maxBytes <= 0 {
		return S3Concurrent{}, errors.New("all parameters must be greater than 0")
	}
	if maxConnections > maxBytes {
		return S3Concurrent{}, errors.New("max bytes must be greater than or equal to max connections")
	}
	if maxConnectionsPerRequest > maxConnections {
		return S3Concurrent{}, errors.New("max connections must be greater than or equal to max connections per request")
	}
	return S3Concurrent{
		S3:      s3Client,
		manager: newConcurrencyManager(maxConnections, maxConnectionsPerRequest, maxBytes),
	}, nil
}

// newConcurrencyManager returns a new ConcurrencyManager set with the given specifications.
func newConcurrencyManager(maxWorkers, maxWorkersPerRequest, maxBytes int) *ConcurrencyManager {
	cm := ConcurrencyManager{}

	// Create worker pool
	wp := make(chan *worker, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		wp <- &worker{
			manager: &cm,
			input:   make(chan types.Object, 1),
			output:  make(chan HydratedFile, 1),
		}
	}
	cm.workerPool = workerPool{channel: wp}

	// Create memory pool. This consists of a channel of "memory chunks",
	// each of which is represented as an int64. The number of chunks and
	// each chunk's size is calculated so that if all workers are downloading a file,
	// the total size of all those files would be less than or equal to
	// the specified max number of bytes.
	mp := make(chan int64, maxWorkers)
	memoryChunkSize := int64(maxBytes / maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		mp <- memoryChunkSize
	}
	cm.memoryPool = memoryPool{channel: mp}
	cm.memoryTotalSize = memoryChunkSize * int64(maxWorkers)
	cm.memoryChunkSize = memoryChunkSize
	cm.maxWorkersPerRequest = maxWorkersPerRequest

	return &cm
}

// GetAllConcurrently gets the objects specified from bucket and writes the resulting HydratedFiles
// to the returned output channel. The closure of this channel is handled, however it's the caller's
// responsibility to purge the channel, and handle any errors present in the HydratedFiles.
// If the ConcurrencyManager is not initialised before calling GetAllConcurrently, an output channel
// containing a single HydratedFile with an error is returned.
// Version can be empty, but must be the same for all objects.
func (s *S3Concurrent) GetAllConcurrently(bucket, version string, objects []types.Object) chan HydratedFile {

	if s.manager == nil {
		output := make(chan HydratedFile, 1)
		output <- HydratedFile{Error: errors.New("error getting files from S3, Concurrency Manager not initialised")}
		close(output)
		return output
	}

	if s.manager.memoryTotalSize < s.manager.calculateRequiredMemoryFor(objects) {
		output := make(chan HydratedFile, 1)
		output <- HydratedFile{Error: fmt.Errorf("error: bytes requested greater than max allowed by server (%v)", s.manager.memoryTotalSize)}
		close(output)
		return output
	}
	// Secure memory for all objects upfront.
	s.manager.secureMemory(objects) // 0.

	processFunc := func(input types.Object) HydratedFile {
		buf := bytes.NewBuffer(make([]byte, 0, int(*input.Size)))
		key := aws.ToString(input.Key)
		err := s.Get(bucket, key, version, buf)

		return HydratedFile{
			Key:   key,
			Data:  buf.Bytes(),
			Error: err,
		}
	}
	return s.manager.Process(processFunc, objects)
}

// getWorker retrieves a number of workers from the manager's worker pool.
func (cm *ConcurrencyManager) getWorkers(number int) []*worker {
	cm.workerPool.mutex.Lock()
	defer cm.workerPool.mutex.Unlock()

	workers := make([]*worker, number)
	for i := 0; i < number; i++ {
		workers[i] = <-cm.workerPool.channel
	}
	return workers
}

// returnWorker returns a worker to the manager's worker pool.
func (cm *ConcurrencyManager) returnWorker(w *worker) {
	cm.workerPool.channel <- w
}

// secureMemory secures the memory needed for the given objects
// from the manager's memory pool.
func (cm *ConcurrencyManager) secureMemory(objects []types.Object) {
	cm.memoryPool.mutex.Lock()
	defer cm.memoryPool.mutex.Unlock()

	for _, o := range objects {
		var securedMemory int64 = 0
		for securedMemory < aws.ToInt64(o.Size) {
			securedMemory += <-cm.memoryPool.channel
		}
	}
}

// calculateRequiredMemoryFor calculates the amount of memory required to contain
// the given objects based on size. Useful as a precheck before securing to
// ensure there's enough in the pool to fulfill the request.
func (cm *ConcurrencyManager) calculateRequiredMemoryFor(objects []types.Object) int64 {
	var totalMemory int64 = 0
	for _, o := range objects {
		numberOfChunks := aws.ToInt64(o.Size) / cm.memoryChunkSize
		if aws.ToInt64(o.Size)%cm.memoryChunkSize != 0 {
			numberOfChunks++
		}
		totalMemory += numberOfChunks * cm.memoryChunkSize
	}
	return totalMemory
}

// releaseMemory returns the specified amount of memory back to
// the manager's memory pool.
func (cm *ConcurrencyManager) releaseMemory(size int64) {
	memoryToRelease := size
	for memoryToRelease > 0 {
		cm.memoryPool.channel <- cm.memoryChunkSize
		memoryToRelease -= cm.memoryChunkSize
	}
}

// Functions for providing a fan-out/fan-in operation. Workers are taken from the
// worker pool and added to a WorkerGroup. All workers are returned to the pool once
// the jobs have finished.
func (cm *ConcurrencyManager) Process(asyncProcessor FileProcessor, objects []types.Object) chan HydratedFile {
	workerGroup := cm.newWorkerGroup(context.Background(), asyncProcessor, cm.maxWorkersPerRequest) // 1.

	go func() {
		for _, obj := range objects {
			workerGroup.addWork(obj)
		}
		workerGroup.stopWork() // 9.
	}()
	return workerGroup.returnOutput() // 2.
}

// start begins a worker's process of making itself available for work, doing the work,
// and repeat, until all work is done.
func (w *worker) start(ctx context.Context, processor FileProcessor, roster chan *worker, wg *sync.WaitGroup) {
	go func() {
		defer func() {
			wg.Done()

			// Make sure workers contents have been consumed
			// before returning to pool.
			if len(w.input) > 0 {
				input := <-w.input
				w.output <- processor(input)
				w.manager.releaseMemory(int64(*input.Size))
			}
			for len(w.output) > 0 {
				time.Sleep(1 * time.Millisecond)
			}

			w.manager.returnWorker(w) // 10.
		}()
		for {
			roster <- w // 3., 7.

			select {
			case input := <-w.input: // 5.
				w.output <- processor(input) // 6.
				w.manager.releaseMemory(int64(*input.Size))
			case <-ctx.Done(): // 9.
				return
			}
		}
	}()
}

type workerGroup struct {
	roster    chan *worker
	reception chan chan HydratedFile
	output    chan HydratedFile
	group     *sync.WaitGroup
	stop      func()
}

// newWorkerGroup creates and returns a new workerGroup, which is a group of workers assembled
// to service a request.
func (cm *ConcurrencyManager) newWorkerGroup(ctx context.Context, processor FileProcessor, size int) workerGroup {
	ctx, cancel := context.WithCancel(ctx)
	workerGroup := workerGroup{
		roster:    make(chan *worker, size),
		reception: make(chan chan HydratedFile, size),
		output:    make(chan HydratedFile),
		stop:      cancel,
		group:     &sync.WaitGroup{},
	}
	workerGroup.group.Add(size)

	go func() {
		workers := cm.getWorkers(size)
		for _, w := range workers {
			w.start(ctx, processor, workerGroup.roster, workerGroup.group)
		}
	}()
	go workerGroup.startOutput()
	go workerGroup.cleanUp(ctx)

	return workerGroup
}

// startOutput begins the process of directing each worker's output
// to the output channel.
func (wg *workerGroup) startOutput() {
	defer close(wg.output)
	for woc := range wg.reception {
		wg.output <- <-woc // 8.
	}
}

// cleanUp blocks on the workerGroup's cancel Context. Once Done(),
// it then waits for the workerGroup's WaitGroup to finish. After that,
// the workerGroup's channels are closed.
func (wg *workerGroup) cleanUp(ctx context.Context) {
	<-ctx.Done()
	wg.group.Wait() // 9.
	close(wg.reception)
	close(wg.roster)
}

// addWork gets the first available worker from the workerGroup's
// roster, and gives it an S3 Object to download. The worker's output
// channel is registered to the workerGroup's reception so that
// order is retained.
func (wg *workerGroup) addWork(newWork types.Object) { // 4.
	for w := range wg.roster {
		w.input <- newWork
		wg.reception <- w.output
		break
	}
}

// returnOutput returns the workerGroup's output channel.
func (wg *workerGroup) returnOutput() chan HydratedFile {
	return wg.output
}

// stopWork calls the workerGroup's stop function, which initiates
// the cleanup process.
func (wg *workerGroup) stopWork() {
	wg.stop()
}
