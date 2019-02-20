package main

import (
	"sync"
	"time"
)

type Analytic struct {
	lock          sync.RWMutex
	Start         time.Time
	End           time.Time
	Channels      map[string]int64
	ChannelsReply map[string]int64
	Users         map[string]int64
	UsersReply    map[string]int64
	FilesNb       int64
	FilesSize     int64
}

func NewAnalytic() *Analytic {
	return &Analytic{
		lock:          sync.RWMutex{},
		Start:         time.Now(),
		Channels:      make(map[string]int64),
		ChannelsReply: make(map[string]int64),
		Users:         make(map[string]int64),
		UsersReply:    make(map[string]int64),
		FilesNb:       int64(0),
		FilesSize:     int64(0),
	}
}

func (a *Analytic) Init() {
	a.Start = time.Now()
	a.End = time.Time{}
	a.Channels = make(map[string]int64)
	a.ChannelsReply = make(map[string]int64)
	a.Users = make(map[string]int64)
	a.UsersReply = make(map[string]int64)
	a.FilesNb = int64(0)
	a.FilesSize = int64(0)

}

func (a *Analytic) WLock() {
	a.lock.Lock()
}

func (a *Analytic) WUnlock() {
	a.lock.Unlock()
}

func (a *Analytic) RLock() {
	a.lock.RLock()
}

func (a *Analytic) RUnlock() {
	a.lock.RUnlock()
}

func (a *Analytic) Close() *Analytic {
	a.End = time.Now()
	return a
}
