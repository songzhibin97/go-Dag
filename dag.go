package go_Dag

import (
	"context"
	"errors"
	"fmt"
	"github.com/songzhibin97/gkit/goroutine"
	"github.com/songzhibin97/gkit/options"
	"sync"
	"sync/atomic"
)

var ErrCircle = errors.New("circle graph")

type Actuator struct {
	Config
	ReadyTask   chan *Task
	SuccessTask chan *Task
	FailureTask chan *Task

	retryLock sync.Mutex

	taskLock sync.Mutex
	taskMap  map[string]*Task

	g goroutine.GGroup

	wg     sync.WaitGroup
	done   int32
	runner int64
}

func (a *Actuator) addRunner() {
	atomic.AddInt64(&a.runner, 1)
}

func (a *Actuator) subRunner() {
	atomic.AddInt64(&a.runner, -1)
}

func (a *Actuator) getRunner() int64 {
	return atomic.LoadInt64(&a.runner)
}

func (a *Actuator) Close() {
	if !atomic.CompareAndSwapInt32(&a.done, 0, 1) {
		return
	}
	close(a.ReadyTask)
}

func (a *Actuator) run() {
	defer a.wg.Done()
	a.wg.Add(3)
	go func() {
		defer a.wg.Done()
		var dist sync.WaitGroup

		for t := range a.ReadyTask {
			func(t *Task) {
				a.addRunner()
				a.wg.Add(1)
				dist.Add(1)
				a.g.AddTask(func() {
					defer a.wg.Done()
					defer a.subRunner()
					defer dist.Done()

					t.setStatue(StateStarted)
					t.Call()
					switch t.GetState() {
					case StateFailure:
						dist.Add(1)
						go func() {
							defer dist.Done()
							a.FailureTask <- t
						}()
					case StateSuccess:
						dist.Add(1)
						go func() {
							defer dist.Done()
							a.SuccessTask <- t
						}()
					default:
						fmt.Printf("unknow state %#v \r\n", t)
					}
				})
			}(t)
		}

		dist.Wait()
		close(a.FailureTask)
		close(a.SuccessTask)
	}()

	go func() {
		defer a.wg.Done()
		for t := range a.FailureTask {
			t.setStatue(StateRetry)
			if a.Retry == nil {
				fmt.Printf("retry is nil, stop! %#v \r\n", t)
				continue
			}
			if !a.Retry(a, t) {
				fmt.Printf("retry is false, stop! %#v \r\n", t)
				fmt.Printf("err: %#v \r\n", t.errors)
				continue
			}
		}
	}()

	go func() {
		defer a.wg.Done()
		for t := range a.SuccessTask {

			list := t.DelOutDegreeRelation()
			if a.getRunner() == 0 && len(list) == 0 {
				// close
				a.Close()
				return
			}
			for _, task := range list {
				a.ReadyTask <- task
			}
		}
	}()
}

func (a *Actuator) reset() {
	for !atomic.CompareAndSwapInt32(&a.done, 1, 0) {
		return
	}
	a.wg.Wait()

	a.ReadyTask = make(chan *Task, a.Config.ChannelBuffer)
	a.SuccessTask = make(chan *Task, a.Config.ChannelBuffer)
	a.FailureTask = make(chan *Task, a.Config.ChannelBuffer)
	a.g = goroutine.NewGoroutine(context.Background(), goroutine.SetMax(int64(a.Config.WorkerNum)))
}

func (a *Actuator) Run() {

	a.reset()

	a.wg.Add(1)
	go a.run()

	a.taskLock.Lock()
	if len(a.taskMap) == 0 {
		a.taskLock.Unlock()
		return
	}

	var clear []*Task
	for _, t := range a.taskMap {
		switch t.GetState() {
		case StateReceived:
			a.ReadyTask <- t
		case StateSuccess:
			clear = append(clear, t)
			delete(a.taskMap, t.GID())
		}
	}

	a.taskLock.Unlock()

	if len(clear) != 0 {
		fmt.Printf("clear: %#v \r\n", clear)
	}
	a.wg.Wait()
}

type Config struct {
	Retry         func(a *Actuator, task *Task) bool
	WorkerNum     int
	ChannelBuffer int
}

func SetRetry(retry func(a *Actuator, task *Task) bool) options.Option {
	return options.Option(func(o interface{}) {
		if retry == nil {
			return
		}
		o.(*Config).Retry = retry
	})
}

func SetWorkerNum(workerNum int) options.Option {
	return options.Option(func(o interface{}) {
		if workerNum <= 0 {
			return
		}
		o.(*Config).WorkerNum = workerNum
	})
}

func SetChannelBuffer(channelBuffer int) options.Option {
	return options.Option(func(o interface{}) {
		if channelBuffer <= 0 {
			return
		}
		o.(*Config).ChannelBuffer = channelBuffer
	})
}

func (a *Actuator) AddTask(task ...*Task) error {
	a.taskLock.Lock()
	defer a.taskLock.Unlock()
	for _, t := range task {
		roots, err := startTask(make(map[string]bool), t, nil)
		if err != nil {
			return err
		}
		for index, root := range roots {
			a.taskMap[root.GID()] = roots[index]
			root.setStatue(StateReceived)
		}
	}
	return nil
}

func startTask(circle map[string]bool, t *Task, root *Task) ([]*Task, error) {
	if root == t {
		return nil, ErrCircle
	}
	if root == nil {
		root = t
	}
	if circle[t.id] {
		return nil, nil
	}
	in := t.allInDegree()
	if len(in) == 0 {
		return []*Task{t}, nil
	}
	circle[t.GID()] = true
	var ret []*Task
	for _, v := range in {
		roots, err := startTask(circle, v, root)
		if err != nil {
			return nil, err
		}
		ret = append(ret, roots...)
	}
	return ret, nil
}

func NewActuator(options ...options.Option) *Actuator {
	c := &Config{
		Retry: func(a *Actuator, task *Task) bool {
			if task.ExecCount() > 3 {
				return false
			}
			task.AddExecCount()
			a.ReadyTask <- task
			return true
		},
		WorkerNum:     1000,
		ChannelBuffer: 1000,
	}
	for _, option := range options {
		option(c)
	}

	a := &Actuator{
		ReadyTask:   make(chan *Task, c.ChannelBuffer),
		SuccessTask: make(chan *Task, c.ChannelBuffer),
		FailureTask: make(chan *Task, c.ChannelBuffer),
		taskMap:     make(map[string]*Task),
		Config:      *c,
		g:           goroutine.NewGoroutine(context.Background(), goroutine.SetMax(int64(c.WorkerNum))),
	}

	return a
}
