package util

import (
	"bore/internal/util/set"
	"sync"
	"time"
)

type Observable interface {
	Register(id interface{}, ch chan<- interface{})
	Unregister(id interface{}, ch chan interface{})
	UnregisterAll(id interface{})

	Notify(id interface{}, data interface{}) bool
	Broadcast(data interface{})

	Clear()
}

type DefaultObservable struct {
	once        bool
	nonBlocking bool
	chanTimeout time.Duration
	observers   *sync.Map
}

type entry struct {
	mu  sync.Mutex
	set *set.Set
}

func NewDefaultObservable(
	once, nonBlocking bool, chanTimeout time.Duration,
) *DefaultObservable {
	return &DefaultObservable{
		once:        once,
		nonBlocking: nonBlocking,
		chanTimeout: chanTimeout,
		observers:   &sync.Map{},
	}
}

func newEntry() *entry {
	return &entry{
		mu:  sync.Mutex{},
		set: set.New(),
	}
}

func (o *DefaultObservable) Register(id interface{}, ch chan<- interface{}) {
	value, _ := o.observers.LoadOrStore(id, newEntry())
	entry := value.(*entry)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.set.Put(ch)
}

func (o *DefaultObservable) UnregisterAll(id interface{}) {
	o.observers.Delete(id)
}

func (o *DefaultObservable) Unregister(id interface{}, ch chan interface{}) {
	if value, ok := o.observers.Load(id); ok {
		entry := value.(*entry)
		entry.mu.Lock()
		defer entry.mu.Unlock()

		entry.set.Remove(ch)
	}
}

func (o *DefaultObservable) Notify(id interface{}, data interface{}) bool {
	if value, ok := o.observers.Load(id); ok {
		o.notify(id, value.(*entry), data)
		return true
	}

	return false
}

func (o *DefaultObservable) Broadcast(data interface{}) {
	o.observers.Range(func(key, value interface{}) bool {
		o.notify(key, value.(*entry), data)
		return true
	})
}

func (o *DefaultObservable) notify(key interface{}, entry *entry, data interface{}) {
	if o.once {
		o.observers.Delete(key)
	}

	entry.mu.Lock()
	channels := entry.set.Array()
	entry.mu.Unlock()

	timeout := o.chanTimeout

	for _, obj := range channels {

		ch := obj.(chan<- interface{})
		if o.nonBlocking {
			go notifyChannel(ch, data, o.once, timeout)
		} else {
			notifyChannel(ch, data, o.once, timeout)
		}
	}
}

func notifyChannel(
	ch chan<- interface{}, data interface{},
	autoClose bool, timeout time.Duration,
) {

	if timeout < 0 {
		ch <- data
	} else if timeout == 0 {

		select {
		case ch <- data:
		default:
		}

	} else {

		select {
		case ch <- data:
		case <-time.After(timeout):
		}

	}

	if autoClose {
		close(ch)
	}
}

func (o *DefaultObservable) Clear() {
	o.observers.Range(func(key, value interface{}) bool {
		o.observers.Delete(key)

		close(value.(chan interface{}))
		return true
	})
}
