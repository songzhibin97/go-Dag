package go_Dag

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

type mock struct {
}

func (mock) Call(t *Task) error {
	time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	fmt.Println(t.id)
	return nil
}

func TestNewTask(t *testing.T) {
	t1, err := NewTask("t1", mock{}, nil)
	assert.NoError(t, err)
	t1.AddError(errors.New("test"))
	assert.Equal(t, 1, len(t1.errors))
	err = t1.SetPayload("test")
	assert.Equal(t, "test", t1.GetPayload())
	assert.Equal(t, "t1", t1.GID())
	assert.Equal(t, StatePending, t1.GetState())
	assert.Equal(t, int64(0), t1.ExecCount())
	t1.AddExecCount()
	assert.Equal(t, int64(1), t1.ExecCount())

	t1.setStatue(StateSuccess)
	assert.Equal(t, StateSuccess, t1.GetState())

	assert.Equal(t, false, t1.IsModify())
	assert.Error(t, t1.SetPayload("t2"))
	assert.Error(t, t1.AddInDegree(nil))
	assert.Error(t, t1.AddInDegree(t1))
	assert.Equal(t, 0, len(t1.inDegree))

	assert.Error(t, t1.AddOutDegree(nil))
	assert.Error(t, t1.AddOutDegree(t1))
	assert.Equal(t, 0, len(t1.outDegree))

	t2, _ := NewTask("t2", mock{}, nil)
	t3, _ := NewTask("t3", mock{}, nil)
	t4, _ := NewTask("t4", mock{}, nil)

	t1.setStatue(StatePending)

	assert.NoError(t, t1.AddOutDegree(t2))
	assert.Equal(t, 1, len(t1.outDegree))
	assert.Equal(t, 1, len(t2.inDegree))

	assert.NoError(t, t1.AddInDegree(t3))
	assert.Equal(t, 1, len(t1.inDegree))
	assert.Equal(t, 1, len(t3.outDegree))

	assert.NoError(t, t1.AddOutDegree(t4))
	assert.Equal(t, 2, len(t1.outDegree))
	assert.Equal(t, 1, len(t4.inDegree))

	t.Log("t1 依赖:", t1.inDegree)
	t.Log("依赖 t1:", t1.outDegree)

	out := t1.DelOutDegreeRelation()
	assert.Equal(t, 2, len(out))
	assert.Equal(t, 0, len(t2.inDegree))
	assert.Equal(t, 0, len(t4.inDegree))

	out = t3.DelOutDegreeRelation()
	assert.Equal(t, 1, len(out))
	assert.Equal(t, 0, len(t2.outDegree))
	assert.Equal(t, 0, len(t2.inDegree))

}
