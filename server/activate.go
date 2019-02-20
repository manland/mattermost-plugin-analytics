package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
	chart "github.com/wcharczuk/go-chart"
)

func (p *Plugin) OnActivate() error {
	teams, err := p.API.GetTeams()
	if err != nil {
		return errors.Wrap(err, "failed to query teams OnActivate")
	}

	for _, team := range teams {
		if err := p.registerCommand(team.Id); err != nil {
			return errors.Wrap(err, "failed to register command")
		}
	}

	if err := p.retreiveData(); err != nil {
		return err
	}
	p.cronSavePoison = make(chan bool)
	p.startCronSaver(p.cronSavePoison)

	return nil
}

func (p *Plugin) registerCommand(teamId string) error {
	if err := p.API.RegisterCommand(&model.Command{
		TeamId:           teamId,
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

	p.cronSavePoison <- true

	return nil
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/line.svg":
		p.handleLine(w, r)
	case "/pie.svg":
		p.handlePie(w, r)
	case "/bar.svg":
		p.handleBar(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleLine(w http.ResponseWriter, r *http.Request) {
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
				v, _ := strconv.ParseFloat(value, 64)
				if v > max {
					max = v
				}
				yvalue = append(yvalue, v)
			}
			yvalues[key] = yvalue
		}
	}
	chartSeries := make([]chart.Series, 0)
	for key, yvalue := range yvalues {
		chartSeries = append(chartSeries,
			chart.TimeSeries{
				Name:    key,
				XValues: times,
				YValues: yvalue,
			},
		)
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
	err := graph.Render(chart.SVG, w)
	if err != nil {
		p.API.LogError("Error rendering line chart", "err", err.Error())
	}
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
			p.API.LogError("Error rendering pie chart", "value", v, "key", key)
		}
	}
	graph := chart.BarChart{
		Width:  300,
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
