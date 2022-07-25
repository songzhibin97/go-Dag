# go-Dag(Directed acyclic graph)任务多线程执行器


## 解决了什么问题
在多任务有依赖的情况下形成 dag(Directed acyclic graph), 通常情况下使用拓扑排序的方式来执行任务,传统拓扑排序会是任务变为串行化执行,
降低任务执行效率

## 如何解决
提交任务将提交的图展开至多个依赖节点,并发执行依赖项(本质还是依赖拓扑结构),在执行完成后将当前节点的出度相关节点的入度在图上删除,判断出度节点是否符合任务执行条件

## 不足
1. 错误处理机制, 目前只有重试机制,目前没有想到怎么做,当前是重试到一定次数舍去,剪枝,但是最终会导致需要执行任务的节点可能无法执行
2. 当前所有数据都是在内存中进行存储,包括crash没有明确处理 
3. 支持的规则较为简单,默认所有的入度条件是 `and` 关系

## 使用

```go
	/*
			  a     b
			/ | \ / |
		   c  d  e   |
			  |    \ /|
			  f --> g |
			         \|
			          h
	*/
	

	
// 建立关系 

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

// 执行 任务 `h`

func main() {
	actuator = NewActuator()
	err = actuator.AddTask(h) // 提交任务
	actuator.Run() // 执行
}


```