package dynamicPool

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewPoolTask(t *testing.T) {
	pool := NewPool(1000, false)

	for i := 0; i < 10000; i++ {
		pool.PushTask(func(i []interface{}) {
			time.Sleep(time.Second)
		}, []interface{}{})
	}
	go func() {
		for {
			pool.getInit()
			time.Sleep(time.Second * 2)
		}
	}()
	time.Sleep(time.Minute * 2)
}

func BenchmarkNewPoolTask(b *testing.B) {
	//b.N = 1000
	{
		b.Run("100 jam", func(b *testing.B) {
			pool := NewPool(100, true)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
		b.Run("1000 jam", func(b *testing.B) {
			pool := NewPool(1000, true)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
		b.Run("10000 jam", func(b *testing.B) {
			pool := NewPool(10000, true)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
		b.Run("100000 jam", func(b *testing.B) {
			pool := NewPool(100000, true)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
	}

	{
		b.Run("100 not jam", func(b *testing.B) {
			pool := NewPool(100, false)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
		b.Run("1000 not jam", func(b *testing.B) {
			pool := NewPool(1000, false)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
		b.Run("10000 not jam", func(b *testing.B) {
			pool := NewPool(10000, false)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
		b.Run("100000 not jam", func(b *testing.B) {
			pool := NewPool(100000, false)
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				pool.PushTask(func(i []interface{}) {
					//b.Log("o => ", i[0], "len", len(i), "type", reflect.ValueOf(i[0]).Type())
					time.Sleep(time.Millisecond * 10)
					wg.Done()
				}, []interface{}{i})
			}
			wg.Wait()
		})
	}
}

func TestNewDynamicPool(t *testing.T) {
	pool := NewPool(100, true)
	var wg sync.WaitGroup
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		pool.PushTask(func(i []interface{}) {
			time.Sleep(time.Millisecond * 10)
			wg.Done()
		}, []interface{}{})
	}
	fmt.Println("push over", time.Now())
	wg.Wait()
	fmt.Println("exec over", time.Now())
}
