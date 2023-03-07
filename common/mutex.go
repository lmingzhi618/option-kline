package common

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	LockedFlag   int32 = 1
	UnlockedFlag int32 = 0
)

type RWMutex struct {
	in     sync.RWMutex
	status *int32
}

func NewRWMutex() *RWMutex {
	status := UnlockedFlag
	return &RWMutex{
		status: &status,
	}
}

func (m *RWMutex) Lock() {
	m.in.Lock()
}

func (m *RWMutex) Unlock() {
	m.in.Unlock()
	atomic.AddInt32(m.status, UnlockedFlag)
}

func (m *RWMutex) TryLock() bool {
	if atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&m.in)), UnlockedFlag, LockedFlag) {
		atomic.AddInt32(m.status, LockedFlag)
		return true
	}
	return false
}

func (m *RWMutex) IsLocked() bool {
	if atomic.LoadInt32(m.status) == LockedFlag {
		return true
	}
	return false
}
