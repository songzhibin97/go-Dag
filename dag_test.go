package go_Dag

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func demoNode(t *testing.T) (*Task, *Task, *Task, *Task, *Task, *Task, *Task, *Task) {
	a, _ := NewTask("a", mock{}, nil)
	b, _ := NewTask("b", mock{}, nil)
	c, _ := NewTask("c", mock{}, nil)
	d, _ := NewTask("d", mock{}, nil)
	e, _ := NewTask("e", mock{}, nil)
	f, _ := NewTask("f", mock{}, nil)
	g, _ := NewTask("g", mock{}, nil)
	h, _ := NewTask("h", mock{}, nil)

	err := a.AddOutDegrees(c, d, e)
	assert.NoError(t, err)
	err = b.AddOutDegrees(e, g, h)
	assert.NoError(t, err)
	err = d.AddOutDegrees(f)
	assert.NoError(t, err)
	err = e.AddOutDegrees(f, g)
	assert.NoError(t, err)
	err = b.AddOutDegrees(h)
	assert.NoError(t, err)
	err = g.AddOutDegrees(f, h)
	assert.NoError(t, err)
	return a, b, c, d, e, f, g, h
}

func TestAddTask(t *testing.T) {
	a, b, c, d, e, f, g, h := demoNode(t)
	actuator := NewActuator()
	err := actuator.AddTask(a)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actuator.taskMap))
	assert.Equal(t, a, actuator.taskMap[a.GID()])

	actuator = NewActuator()
	err = actuator.AddTask(b)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actuator.taskMap))
	assert.Equal(t, b, actuator.taskMap[b.GID()])

	actuator = NewActuator()
	err = actuator.AddTask(c)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actuator.taskMap))
	assert.Equal(t, a, actuator.taskMap[a.GID()])

	actuator = NewActuator()
	err = actuator.AddTask(d)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actuator.taskMap))
	assert.Equal(t, a, actuator.taskMap[a.GID()])

	actuator = NewActuator()
	err = actuator.AddTask(e)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actuator.taskMap))
	assert.Equal(t, a, actuator.taskMap[a.GID()])
	assert.Equal(t, b, actuator.taskMap[b.GID()])

	actuator = NewActuator()
	err = actuator.AddTask(f)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actuator.taskMap))
	assert.Equal(t, a, actuator.taskMap[a.GID()])
	assert.Equal(t, b, actuator.taskMap[b.GID()])

	actuator = NewActuator()
	err = actuator.AddTask(g)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actuator.taskMap))
	assert.Equal(t, a, actuator.taskMap[a.GID()])
	assert.Equal(t, b, actuator.taskMap[b.GID()])

	actuator = NewActuator(SetWorkerNum(1))
	err = actuator.AddTask(h)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actuator.taskMap))
	assert.Equal(t, a, actuator.taskMap[a.GID()])
	assert.Equal(t, b, actuator.taskMap[b.GID()])
	actuator.Run()
	time.Sleep(time.Second)
}
