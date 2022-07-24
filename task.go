package go_Dag

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrTaskModify = errors.New("task is not modify")
	ErrTaskNil    = errors.New("task is nil")
)

type CallFunc interface {
	Call(task *Task) error
}

type Task struct {
	id            string
	state         State // 任务状态
	inDegreeLock  sync.Mutex
	inDegree      map[string]*Task // 入度
	outDegreeLock sync.Mutex
	outDegree     map[string]*Task // 出度
	call          CallFunc         // 执行函数
	payload       atomic.Value     // 任务参数
	execCount     int64            // 执行次数
	errors        []error          // 错误
}

func (t *Task) setStatue(state State) {
	t.state = state
}

func (t *Task) IsModify() bool {
	return t.state == StatePending
}

func (t *Task) AddError(err error) {
	t.errors = append(t.errors, err)
}

func (t *Task) SetPayload(payload interface{}) error {
	if payload == nil {
		return nil
	}
	if !t.IsModify() {
		return ErrTaskModify
	}
	t.payload.Store(payload)
	return nil
}

func (t *Task) GetPayload() interface{} {
	return t.payload.Load()
}

func (t *Task) AddInDegrees(tt ...*Task) error {
	for _, task := range tt {
		err := t.AddInDegree(task)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (t *Task) AddInDegree(tt *Task) error {
	if tt == nil {
		return ErrTaskNil
	}
	if !t.IsModify() || !tt.IsModify() {
		return ErrTaskModify
	}
	// 添加被依赖节点
	tt.outDegreeLock.Lock()
	if tt.outDegree == nil {
		tt.outDegree = make(map[string]*Task)
	}
	tt.outDegree[t.id] = t
	tt.outDegreeLock.Unlock()

	t.inDegreeLock.Lock()
	if t.inDegree == nil {
		t.inDegree = make(map[string]*Task)
	}
	t.inDegree[tt.id] = tt
	t.inDegreeLock.Unlock()
	return nil
}

func (t *Task) AddOutDegrees(tt ...*Task) error {
	for _, task := range tt {
		err := t.AddOutDegree(task)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Task) AddOutDegree(tt *Task) error {
	if tt == nil {
		return ErrTaskNil
	}
	return tt.AddInDegree(t)
}

func (t *Task) GID() string {
	return t.id
}

func (t *Task) GetState() State {
	return t.state
}

func (t *Task) Call() {
	err := t.call.Call(t)
	if err != nil {
		t.errors = append(t.errors, err)
		t.setStatue(StateFailure)
		return
	}
	t.setStatue(StateSuccess)
}

func (t *Task) ExecCount() int64 {
	return atomic.LoadInt64(&t.execCount)
}

func (t *Task) AddExecCount() {
	atomic.AddInt64(&t.execCount, 1)
}

func (t *Task) allInDegree() []*Task {
	t.inDegreeLock.Lock()
	defer t.inDegreeLock.Unlock()
	nodes := make([]*Task, 0, len(t.inDegree))
	for _, n := range t.inDegree {
		nodes = append(nodes, n)
	}
	return nodes
}

func (t *Task) allOutDegree() []*Task {
	t.outDegreeLock.Lock()
	defer t.outDegreeLock.Unlock()
	nodes := make([]*Task, 0, len(t.outDegree))
	for _, n := range t.outDegree {
		nodes = append(nodes, n)
	}
	return nodes
}

func (t *Task) DelInDegreeRelation(id string) bool {
	t.inDegreeLock.Lock()
	defer t.inDegreeLock.Unlock()
	delete(t.inDegree, id)
	return len(t.inDegree) == 0
}

func (t *Task) DelOutDegreeRelation() []*Task {
	nodes := t.allOutDegree()
	next := make([]*Task, 0, len(nodes))
	for index, n := range nodes {
		if !n.DelInDegreeRelation(t.id) {
			continue
		}
		next = append(next, nodes[index])
	}
	return next
}

func NewTask(id string, call CallFunc, payload interface{}) (*Task, error) {
	if call == nil {
		return nil, errors.New("call is nil")
	}
	t := &Task{
		id:    id,
		state: StatePending,
		call:  call,
	}
	if payload != nil {
		t.payload.Store(payload)
	}
	return t, nil
}
