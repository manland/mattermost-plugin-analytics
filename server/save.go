package main

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func (p *Plugin) retreiveData() error {
	j, err := p.API.KVGet("analytics")
	if err != nil {
		return errors.Wrap(err, "failed to get analytics from kv")
	}
	p.currentAnalytic = NewAnalytic()
	if err := json.Unmarshal(j, p.currentAnalytic); err != nil {
		p.API.LogError("failed to unmarshal analytics from kv use new one", "err", err.Error())
		p.currentAnalytic = NewAnalytic()
	}
	return nil
}

func (p *Plugin) saveCurrentAnalytic() error {
	p.currentAnalytic.RLock()
	defer p.currentAnalytic.RUnlock()

	j, err := json.Marshal(p.currentAnalytic)
	if err != nil {
		return errors.Wrap(err, "can't marshal internal analytics data")
	}
	if err := p.API.KVSet("analytics", j); err != nil {
		return errors.Wrap(err, "can't save analytics data")
	}
	return nil
}

func (p *Plugin) allSessions() ([]*Analytic, error) {
	allAnalytics := make([]*Analytic, 0)

	j, err := p.API.KVGet("allAnalytics")
	if err != nil {
		p.API.LogError("can't get allAnalytics", "err", err.Error())
		return nil, err
	}

	if err := json.Unmarshal(j, &allAnalytics); err != nil {
		p.API.LogError("failed to unmarshal analytics from kv use new one", "err", err.Error())
		return nil, err
	}

	return allAnalytics, nil
}

func (p *Plugin) newSession() {
	p.currentAnalytic.WLock()
	defer p.currentAnalytic.WUnlock()

	allAnalytics, err := p.allSessions()
	if err != nil {
		p.API.LogWarn("can't get all sessions", "err", err.Error())
	}

	j2, err2 := json.Marshal(append(allAnalytics, p.currentAnalytic.Close()))
	if err2 != nil {
		p.API.LogWarn("can't marshal internal analytics data")
	}
	if err := p.API.KVSet("allAnalytics", j2); err != nil {
		p.API.LogError("failed to send allAnalytics to kv", "err", err.Error())
	}
	p.currentAnalytic.Init()
}
