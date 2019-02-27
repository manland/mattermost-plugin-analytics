package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
	chart "github.com/wcharczuk/go-chart"
)

// OnActivate is called by mattermost when this plugin is started
func (p *Plugin) OnActivate() error {
	teams, errApp := p.API.GetTeamsForUser(p.BotUserID)
	if errApp != nil {
		return errors.Wrap(errApp, "failed to query teams OnActivate")
	}

	for _, team := range teams {
		if err := p.registerCommand(team.Id); err != nil {
			return errors.Wrap(err, "failed to register command")
		}
	}

	if err := p.retreiveData(); err != nil {
		return err
	}

	c, err := NewCron(p)
	if err != nil {
		return err
	}
	p.cron = c

	return nil
}

func (p *Plugin) registerCommand(teamID string) error {
	if err := p.API.RegisterCommand(&model.Command{
		TeamId:           teamID,
		Trigger:          CommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Display analytics of this channel",
		DisplayName:      "Analytics of this channel",
		Description:      "A command used to show analytics of this channel.",
	}); err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	return nil
}

// OnDeactivate is called by mattermost when this plugin is deactivated
func (p *Plugin) OnDeactivate() error {
	teams, err := p.API.GetTeams()
	if err != nil {
		return errors.Wrap(err, "failed to query teams OnDeactivate")
	}

	for _, team := range teams {
		if err := p.API.UnregisterCommand(team.Id, CommandTrigger); err != nil {
			return errors.Wrap(err, "failed to unregister command")
		}
	}

	p.cron.Stop()

	return nil
}

// ServeHTTP is called by mattermost when an http request is made to this plugin
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.URL.Path {
	case "/line.svg":
		err = p.handleLine(w, r)
	case "/pie.svg":
		p.handlePie(w, r)
	case "/bar.svg":
		p.handleBar(w, r)
	default:
		http.NotFound(w, r)
	}
	if err != nil {
		p.API.LogError("Error rendering chart", "err", err.Error())
	}
}

func (p *Plugin) handleLine(w http.ResponseWriter, r *http.Request) error {
	times := make([]time.Time, 0)
	yvalues := make(map[string][]float64, 0)
	max := -1.0
	for key, values := range r.URL.Query() {
		if key == "amp" {
			continue
		} else if key == "date" {
			for _, stringDate := range values {
				i, _ := strconv.ParseInt(stringDate, 10, 64)
				t := time.Unix(i, 0)
				times = append(times, t)
			}
		} else {
			yvalue := make([]float64, 0)
			for _, value := range values {
				v, err := strconv.ParseFloat(value, 64)
				if err != nil {
					p.API.LogError("can't parse value", "value", value, "url", r.URL.String())
					v = 0
				}
				if v > max {
					max = v
				}
				yvalue = append(yvalue, v)
			}
			yvalues[key] = yvalue
		}
	}

	if len(times) < 2 {
		return fmt.Errorf("Not enought time to draw a chart %d for url %s", len(times), r.URL.String())
	}

	chartSeries := make([]chart.Series, 0)
	for key, yvalue := range yvalues {
		nbYValue := len(yvalue)
		nbTimes := len(times)
		if nbYValue == nbTimes {
			chartSeries = append(chartSeries,
				chart.TimeSeries{
					Name:    key,
					XValues: times,
					YValues: yvalue,
				},
			)
		} else {
			p.API.LogDebug("Not enought data to draw line", "name", key, "nbTimes", nbTimes, "nbData", nbYValue, "url", r.URL.String())
		}
	}

	if len(chartSeries) < 1 {
		return fmt.Errorf("Not enought data to draw a chart %d for url %s", len(chartSeries), r.URL.String())
	}

	graph := chart.Chart{
		Width:  800,
		Height: 300,
		XAxis: chart.XAxis{
			Style: chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Style: chart.StyleShow(),
			Range: &chart.ContinuousRange{Min: 0, Max: max},
		},
		Series: chartSeries,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	w.Header().Set("Content-Type", chart.ContentTypeSVG)
	return graph.Render(chart.SVG, w)
}

func (p *Plugin) handlePie(w http.ResponseWriter, r *http.Request) {
	values := make([]chart.Value, 0)
	for key, value := range r.URL.Query() {
		if key == "amp" {
			continue
		}
		v, _ := strconv.ParseFloat(value[0], 64)
		values = append(values, chart.Value{Value: v, Label: key})
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].Label < values[j].Label
	})
	graph := chart.PieChart{
		Width:  300,
		Height: 300,
		Values: values,
	}

	w.Header().Set("Content-Type", chart.ContentTypeSVG)
	err := graph.Render(chart.SVG, w)
	if err != nil {
		p.API.LogError("Error rendering pie chart", "err", err.Error())
	}
}

func (p *Plugin) handleBar(w http.ResponseWriter, r *http.Request) {
	values := make([]chart.Value, 0)
	max := -1.0
	for key, value := range r.URL.Query() {
		if key != "amp" {
			v, _ := strconv.ParseFloat(value[0], 64)
			if v > max {
				max = v
			}
			values = append(values, chart.Value{Value: v, Label: key})
		}
	}
	graph := chart.BarChart{
		Width:  600,
		Height: 300,
		XAxis:  chart.StyleShow(),
		YAxis: chart.YAxis{
			Style: chart.StyleShow(),
			Range: &chart.ContinuousRange{Min: 0, Max: max},
		},
		Bars: values,
	}

	w.Header().Set("Content-Type", chart.ContentTypeSVG)
	err := graph.Render(chart.SVG, w)
	if err != nil {
		p.API.LogError("Error rendering bar chart", "err", err.Error())
	}
}
