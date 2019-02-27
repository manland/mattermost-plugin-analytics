package main

import (
	"github.com/robfig/cron"
)

// Cron manage all cron jobs of this plugin
// behind the scene it's a facade to github.com/robfig/cron
type Cron struct {
	p *Plugin
	c *cron.Cron
}

// NewCron return a cron
func NewCron(p *Plugin) (*Cron, error) {
	c := cron.New()

	if err := c.AddFunc("@every 1m", func() { // Run once a week, midnight between Sat/Sun
		if err := p.saveCurrentAnalytic(); err != nil {
			p.API.LogError("can't save current analytic", "err", err.Error())
		}
	}); err != nil {
		return nil, err
	}

	if err := c.AddFunc("@weekly", func() { // Run once a week, midnight between Sat/Sun
		if err := p.sendAnalytics(); err != nil {
			p.API.LogError("can't send post", "err", err.Error())
		}
		p.newSession()
	}); err != nil {
		return nil, err
	}

	c.Start()

	return &Cron{
		p: p,
		c: c,
	}, nil
}

// Stop the cron task and save data
func (c *Cron) Stop() {
	if err := c.p.saveCurrentAnalytic(); err != nil {
		c.p.API.LogError("can't save current analytic", "err", err.Error())
	}
	c.c.Stop()
}
