package main

import (
	"sync"
	"time"
)

// Time after which watchdog will revert system to a safe state if haven't had any contact (seconds)
const watchdogTimeout = 10800 // 3h

type watchdog struct {
	mu          sync.RWMutex
	lastContact int64
	running     bool
}

func NewWatchdog(callback func()) (wd *watchdog) {
	wd = &watchdog{
		lastContact: time.Now().Unix(),
		running:     true,
	}

	go func() {
		for wd.running {
			wd.mu.RLock()
			delta := time.Now().Unix() - wd.lastContact
			wd.mu.RUnlock()
			if delta > watchdogTimeout {
				callback()
				wd.Check()
			}
			time.Sleep(watchdogTimeout / 10 * time.Second)
		}
	}()

	return
}

func (wd *watchdog) Close() error {
	wd.running = false
	return nil
}

func (wd *watchdog) Check() {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	wd.lastContact = time.Now().Unix()
}
