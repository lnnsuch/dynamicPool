package dynamicPool

import (
	"container/list"
	"context"
	"sync"
	"time"
)

const workClearInterval = time.Second * 10

type task func([]interface{})

type sendFun struct {
	f      task
	params []interface{}
}

func newSendFun(f task, params []interface{}) *sendFun {
	return &sendFun{f, params}
}

type worker struct {
	pool   Pool
	task   chan *sendFun
	ctx    context.Context
	cancel context.CancelFunc
}

func newWorker(p Pool) *worker {
	ctx, cancel := context.WithCancel(p.getCtx())
	work := &worker{
		pool:   p,
		task:   make(chan *sendFun, 1),
		ctx:    ctx,
		cancel: cancel,
	}
	go work.run()
	return work
}

func (w *worker) run() {
	timer := time.NewTimer(workClearInterval)
	for {
		select {
		case <-w.ctx.Done():
			timer.Stop()
			return
		case v := <-w.task:
			v.f(v.params)
			w.pool.putTask(w)
			timer.Reset(workClearInterval)
		case <-timer.C:
			w.pool.clearTask()
		}
	}
}

type Pool interface {
	getWork() *worker
	putTask(work *worker)
	PushTask(f task, params []interface{})
	clearTask()
	Cancel()
	getCtx() context.Context
}

func NewPool(maxSize uint32, isJam bool) Pool {
	p := newDynamicPool(maxSize)
	if isJam {
		return newDynamicPoolJam(p)
	} else {
		return newDynamicPoolNotJam(p)
	}
}

type dynamicPool struct {
	capacity uint32    // 最大开启的任务数
	running  uint32    // 当前运行的任务数
	workers  []*worker // 可复用的任务
	lock     sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func newDynamicPool(maxSize uint32) *dynamicPool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &dynamicPool{
		capacity: maxSize,
		ctx:      ctx,
		cancel:   cancel,
	}
	return p
}

func (d *dynamicPool) clearTask() {
	d.lock.Lock()
	clearLen := len(d.workers)
	clearWorkers := d.workers
	d.workers = d.workers[:0:0]
	d.running -= uint32(clearLen)
	d.lock.Unlock()
	for _, v := range clearWorkers {
		v.cancel()
	}
}

func (d *dynamicPool) Cancel() {
	d.cancel()
}

func (d *dynamicPool) getCtx() context.Context {
	return d.ctx
}

type dynamicPoolJam struct {
	*dynamicPool
	freeSign *sync.Cond // 任务处理完成发送信号
}

func newDynamicPoolJam(p *dynamicPool) *dynamicPoolJam {
	pool := &dynamicPoolJam{
		dynamicPool: p,
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
			w = newWorker(p)
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
	select {
	case <-p.ctx.Done():
		return
	default:
		p.getWork().task <- newSendFun(f, params)
	}
}

type dynamicPoolNotJam struct {
	*dynamicPool
	pending struct{
		list *list.List
		mux sync.Mutex
	}
}

func newDynamicPoolNotJam(p *dynamicPool) *dynamicPoolNotJam {
	pool := &dynamicPoolNotJam{
		dynamicPool: p,
	}
	pool.pending.list = list.New()
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
		w = newWorker(p)
	}
	return w
}

func (p *dynamicPoolNotJam) putTask(work *worker) {
	p.pending.mux.Lock()
	front := p.pending.list.Front()
	if front == nil {
		p.pending.mux.Unlock()
		p.lock.Lock()
		p.workers = append(p.workers, work)
		p.lock.Unlock()
	} else {
		fun := p.pending.list.Remove(front).(*sendFun)
		p.pending.mux.Unlock()
		work.task <- fun
	}
}

func (p *dynamicPoolNotJam) PushTask(f task, params []interface{}) {
	select {
	case <-p.ctx.Done():
		return
	default:
		fun := newSendFun(f, params)
		work := p.getWork()
		if work == nil {
			p.pending.mux.Lock()
			p.pending.list.PushBack(fun)
			p.pending.mux.Unlock()
		} else {
			work.task <- fun
		}
	}
}
