package dynamicPool

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

type task func([]interface{})

type sendFun struct {
	f      task
	params []interface{}
}

type worker struct {
	pool Pool
	task chan *sendFun
}

func (w *worker) run() {
	go func() {
		for v := range w.task {
			v.f(v.params)
			w.pool.putTask(w)
		}
	}()
}

func (w *worker) pending() {
	pending := list.New()
	var mux sync.Mutex
	go func() {
		for {
			mux.Lock()
			front := pending.Front()
			if front == nil {
				mux.Unlock()
				time.Sleep(time.Millisecond * 10)
				continue
			}
			fun := pending.Remove(front).(*sendFun)
			mux.Unlock()
			w.pool.PushTask(fun.f, fun.params)
		}
	}()
	for {
		select {
		case fun := <-w.task:
			mux.Lock()
			pending.PushBack(fun)
			mux.Unlock()
		}
	}
}

type Pool interface {
	getWork() *worker
	putTask(work *worker)
	PushTask(f task, params []interface{})
}

func NewPool(maxSize uint32, isJam bool) Pool {
	if isJam {
		return newDynamicPoolJam(maxSize)
	} else {
		return newDynamicPoolNotJam(maxSize)
	}
}

type dynamicPool struct {
	capacity uint32    // 最大开启的任务数
	running  uint32    // 当前运行的任务数
	workers  []*worker // 可复用的任务
	lock     sync.Mutex
	once     sync.Once
	clear    chan struct{}
}

func newDynamicPool(maxSize uint32) *dynamicPool {
	p := &dynamicPool{
		capacity: maxSize,
		clear:    make(chan struct{}),
	}
	go p.listenClear()
	return p
}

func (d *dynamicPool) listenClear() {
	tick := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-d.clear:
		case <-tick.C:
			d.lock.Lock()
			clearLen := len(d.workers)
			d.workers = d.workers[:0:0]
			d.running -= uint32(clearLen)
			d.lock.Unlock()
		}
	}
}

type dynamicPoolJam struct {
	*dynamicPool
	freeSign *sync.Cond // 任务处理完成发送信号
}

func newDynamicPoolJam(maxSize uint32) *dynamicPoolJam {
	pool := &dynamicPoolJam{
		dynamicPool: newDynamicPool(maxSize),
	}
	pool.freeSign = sync.NewCond(&pool.lock)
	return pool
}

func (p *dynamicPoolJam) getWork() *worker {
	var w *worker
	p.lock.Lock()
	defer p.lock.Unlock()
	for {
		if len(p.workers) > 0 {
			w = p.workers[0]
			p.workers = p.workers[1:]
		} else if p.capacity > p.running {
			p.running++
			w = &worker{
				pool: p,
				task: make(chan *sendFun),
			}
			w.run()
		}
		if w != nil {
			return w
		}
		p.freeSign.Wait()
	}
}

func (p *dynamicPoolJam) putTask(work *worker) {
	p.lock.Lock()
	p.workers = append(p.workers, work)
	p.lock.Unlock()
	p.freeSign.Signal()
}

func (p *dynamicPoolJam) PushTask(f task, params []interface{}) {
	p.clear <- struct{}{}
	p.getWork().task <- &sendFun{f, params}
}

type dynamicPoolNotJam struct {
	*dynamicPool
	waitTask *worker
}

func newDynamicPoolNotJam(maxSize uint32) *dynamicPoolNotJam {
	pool := &dynamicPoolNotJam{
		dynamicPool: newDynamicPool(maxSize),
	}
	pool.waitTask = &worker{
		pool: pool,
		task: make(chan *sendFun),
	}
	go pool.waitTask.pending()
	return pool
}

func (p *dynamicPoolNotJam) getWork() *worker {
	var w *worker
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.workers) > 0 {
		w = p.workers[0]
		p.workers = p.workers[1:]
	} else if p.capacity > p.running {
		p.running++
		w = &worker{
			pool: p,
			task: make(chan *sendFun),
		}
		w.run()
	}
	if w != nil {
		return w
	}
	return p.waitTask
}

func (p *dynamicPoolNotJam) putTask(work *worker) {
	p.lock.Lock()
	p.workers = append(p.workers, work)
	p.lock.Unlock()
}

func (p *dynamicPoolNotJam) PushTask(f task, params []interface{}) {
	p.clear <- struct{}{}
	p.getWork().task <- &sendFun{f, params}
}
