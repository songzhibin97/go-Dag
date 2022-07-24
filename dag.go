package go_Dag

import (
	"context"
	"errors"
	"fmt"
	"github.com/songzhibin97/gkit/goroutine"
	"github.com/songzhibin97/gkit/options"
	"sync"
)

var ErrCircle = errors.New("circle graph")

type Actuator struct {
	ReadyTask   chan *Task
	SuccessTask chan *Task
	FailureTask chan *Task

	retryLock sync.Mutex

	taskLock sync.Mutex
	taskMap  map[string]*Task

	g goroutine.GGroup

	Config
}

func (a *Actuator) run() {
	go func() {
		for t := range a.ReadyTask {
			func(t *Task) {
				a.g.AddTask(func() {
					t.setStatue(StateStarted)
					t.Call()
					switch t.GetState() {
					case StateFailure:
						a.FailureTask <- t // 如果失败，则把任务放入失败队列
					case StateSuccess:
						go func() {
							a.SuccessTask <- t // 如果成功，则把任务放入成功队列
						}()
					default:
						fmt.Printf("unknow state %#v \r\n", t)
					}
				})
			}(t)
		}
	}()

	go func() {
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
		for t := range a.SuccessTask {
			list := t.DelOutDegreeRelation()
			for _, task := range list {
				a.ReadyTask <- task
			}
		}
	}()

}

// TODO 并发控制(任务的最小执行)

func (a *Actuator) Run() []*Task {
	a.taskLock.Lock()
	defer a.taskLock.Unlock()
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
	return clear
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
	go a.run()

	return a
}
