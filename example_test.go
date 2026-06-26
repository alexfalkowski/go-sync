package sync_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/alexfalkowski/go-sync"
)

func ExampleWait() {
	err := sync.Wait(context.Background(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return nil
		},
	})

	fmt.Println(err == nil)
	// Output: true
}

func ExampleTimeout() {
	err := sync.Timeout(context.Background(), 10*time.Millisecond, sync.Hook{
		OnRun: func(ctx context.Context) error {
			<-ctx.Done()
			return context.Cause(ctx)
		},
	})

	fmt.Println(sync.IsTimeoutError(err))
	// Output: true
}

func ExampleWorker() {
	worker := sync.NewWorker(2)
	var count sync.Int32

	for range 3 {
		err := worker.Schedule(context.Background(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				count.Add(1)
				return nil
			},
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	worker.Wait()
	fmt.Println(count.Load())
	// Output: 3
}

func ExampleWorker_TrySchedule() {
	worker := sync.NewWorker(1)
	var count sync.Int32

	err := worker.TrySchedule(context.Background(), sync.Hook{
		OnRun: func(context.Context) error {
			count.Add(1)
			return nil
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	worker.Wait()
	fmt.Println(count.Load())
	// Output: 1
}

func ExampleErrorGroup() {
	var g sync.ErrorGroup

	g.Go(func() error { return nil })
	g.Go(func() error { return context.Canceled })

	fmt.Println(g.Wait() != nil)
	// Output: true
}

func ExampleErrorsGroup() {
	var g sync.ErrorsGroup

	first := errors.New("first")
	second := errors.New("second")

	g.Go(func() error { return first })
	g.Go(func() error { return second })

	err := g.Wait()
	fmt.Println(errors.Is(err, first), errors.Is(err, second))
	// Output: true true
}

func ExampleSingleFlightGroup() {
	var g sync.SingleFlightGroup[int]

	v, err, shared := g.Do("key", func() (int, error) {
		return 42, nil
	})

	fmt.Println(v, err == nil, shared)
	// Output: 42 true false
}

func ExampleSingleFlightGroup_DoChan() {
	var g sync.SingleFlightGroup[int]

	ch := g.DoChan("key", func() (int, error) {
		return 42, nil
	})
	result := <-ch

	fmt.Println(result.Value, result.Err == nil, result.Shared)
	// Output: 42 true false
}

func ExampleBufferPool() {
	pool := sync.NewBufferPool()
	buffer := pool.Get()
	defer pool.Put(buffer)

	buffer.WriteString("hello")
	copy := pool.Copy(buffer)
	fmt.Println(string(copy))
	// Output: hello
}

func ExamplePool() {
	type item struct {
		id int
	}

	pool := sync.NewPool[item]()
	v := pool.Get()
	v.id = 10
	pool.Put(v)

	v2 := pool.Get()
	fmt.Println(v2 != nil)
	pool.Put(v2)
	// Output: true
}

func ExampleValue() {
	var value sync.Value[int]
	fmt.Println(value.Load())

	value.Store(1)
	fmt.Println(value.Swap(2))
	// Output:
	// 0
	// 1
}

func ExampleMap() {
	var m sync.Map[string, int]
	m.Store("one", 1)

	v, ok := m.Load("one")
	fmt.Println(v, ok)
	// Output: 1 true
}

func ExampleMap_nilInterfaceValue() {
	var m sync.Map[string, io.Reader]
	var r io.Reader
	m.Store("reader", r)

	m.Range(func(_ string, value io.Reader) bool {
		fmt.Println(value == nil)
		return true
	})
	// Output: true
}

func ExampleBufferPool_Copy_nil() {
	pool := sync.NewBufferPool()
	fmt.Println(pool.Copy(nil) == nil)
	// Output: true
}

func ExampleBufferPool_Get() {
	pool := sync.NewBufferPool()
	buffer := pool.Get()
	defer pool.Put(buffer)

	buffer.WriteString("aaa")
	fmt.Println(buffer.String())
	// Output: aaa
}
