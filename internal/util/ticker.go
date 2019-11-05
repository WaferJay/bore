package util

import (
	"errors"
	"time"
)

type Ticker struct {
	C        chan *Ticker
	timeout  time.Duration
	callback func(*Ticker)
	event    chan byte
	running  bool
}

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrAlreadyClosed  = errors.New("already closed")
)

const (
	eventClose = 0
	eventReset = 1
)

func NewTicker(cb func(*Ticker), timeout time.Duration) *Ticker {
	return &Ticker{
		timeout:  timeout,
		callback: cb,
		event:    make(chan byte),
		running:  false,
	}
}

func (t *Ticker) worker() {
	for {
		timer := time.NewTimer(t.timeout)
		select {
		case e := <-t.event:
			switch e {
			case eventReset:
			case eventClose:
				close(t.C)
				return
			}

		case <-timer.C:
			if t.callback != nil {
				go t.callback(t)
			}

			select {
			case t.C <- t:
			default:
			}
		}
	}
}

func (t *Ticker) Start() error {
	if !t.running {
		t.running = true
		t.C = make(chan *Ticker)
		go t.worker()

		return nil
	}
	return ErrAlreadyStarted
}

func (t *Ticker) Reset() error {
	if t.running {
		t.event <- eventReset
		return nil
	}
	return ErrAlreadyClosed
}

func (t *Ticker) Close() error {
	if t.running {
		t.running = false
		t.event <- eventClose
		return nil
	}
	return ErrAlreadyClosed
}
