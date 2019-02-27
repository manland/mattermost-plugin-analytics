package main

import (
	"sync"
	"time"
)

// Analytic is a session of metrics to generate a report.
// See `NewAnalytic()` to build one
type Analytic struct {
	lock sync.RWMutex
	// Start of recording
	Start time.Time
	// End of recording metrics (time.Zero by default)
	End time.Time
	// Channels store number of messages by channels id
	Channels map[string]int64
	// ChannelsReply store number of reply by channels id
	ChannelsReply map[string]int64
	// Users store number of messages by user id
	Users map[string]int64
	// UsersReply store number of reply by user id
	UsersReply map[string]int64
	// FilesNb store number of files uploaded
	FilesNb int64
	// FilesSize store weigth of files uploaded
	FilesSize int64
}

// NewAnalytic return a struct to store all data needed to generate a report
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

// Init reinitialize an analytic to zero
// TODO : remove me if possible
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

// WLock to lock this analytic in write
func (a *Analytic) WLock() {
	a.lock.Lock()
}

// WUnlock to unlock this analytic in write
func (a *Analytic) WUnlock() {
	a.lock.Unlock()
}

// RLock to lock this analytic in read
func (a *Analytic) RLock() {
	a.lock.RLock()
}

// RUnlock to unlock this analytic in read
func (a *Analytic) RUnlock() {
	a.lock.RUnlock()
}

// Close this analytic by adding an End date
func (a *Analytic) Close() *Analytic {
	a.End = time.Now()
	return a
}
