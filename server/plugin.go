package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

// Plugin is the main struct used by mattermost to interact with this plugin
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	currentAnalytic *Analytic

	cronSavePoison chan bool
}

// CommandTrigger is the string used by user to interact with this plugin
const CommandTrigger = "analytics"

// ExecuteCommand will be called by mattermost when user use /analytics command
// used to send a report
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !strings.HasPrefix(args.Command, "/"+CommandTrigger) {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Unknown command: " + args.Command),
		}, nil
	}

	if strings.Contains(args.Command, "new") {
		p.newSession()
		allSessions, _ := p.allSessions()
		j2, _ := json.Marshal(allSessions)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			Text:         string(j2),
		}, nil
	}

	if strings.Contains(args.Command, "session") {
		allSessions, _ := p.allSessions()
		j2, _ := json.Marshal(allSessions)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			Text:         string(j2),
		}, nil
	}

	if strings.Contains(args.Command, "raw") {
		j2, _ := json.Marshal(p.currentAnalytic)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
			Text:         string(j2),
		}, nil
	}

	data, err := p.prepareData()
	if err != nil {
		return nil, &model.AppError{Message: err.Error()}
	}

	p.currentAnalytic.RLock()
	m := fmt.Sprintf("## Analytics since %s at %s\n", p.currentAnalytic.Start.Format("2 January"), p.currentAnalytic.Start.Format("15:04"))
	p.currentAnalytic.RUnlock()
	if data.totalMessagesPublic+data.totalMessagesPrivate > 0 {
		m = m + fmt.Sprintf("#### **%d users** sent a total of **%d messages** in **%d channels**. With **%d** *(%d%%)* in public channels and **%d** *(%d%%)* in private.\n", len(data.users), data.totalMessagesPublic+data.totalMessagesPrivate, len(data.channels), data.totalMessagesPublic, (data.totalMessagesPublic*100)/(data.totalMessagesPublic+data.totalMessagesPrivate), data.totalMessagesPrivate, (data.totalMessagesPrivate*100)/(data.totalMessagesPublic+data.totalMessagesPrivate))
		m = m + fmt.Sprintf("#### And they sent a total of **%d files** for a total of **%s**.\n", p.currentAnalytic.FilesNb, byteCountDecimal(p.currentAnalytic.FilesSize))

		m = m + "### :speak_no_evil: Podium Speaker Users\n"
		if len(data.users) > 0 {
			m = m + fmt.Sprintf("* :1st_place_medal: @%s with a total of **%d** *(%d%%)* public messages with %d reply\n", data.users[0].name, data.users[0].nb, (data.users[0].nb*100)/data.totalMessagesPublic, data.users[0].reply)
		}
		if len(data.users) > 1 {
			m = m + fmt.Sprintf("* :2nd_place_medal: @%s with a total of **%d** *(%d%%)* public messages with %d reply\n", data.users[1].name, data.users[1].nb, (data.users[1].nb*100)/data.totalMessagesPublic, data.users[1].reply)
		}
		if len(data.users) > 2 {
			m = m + fmt.Sprintf("* :3rd_place_medal: @%s with a total of **%d** *(%d%%)* public messages with %d reply\n", data.users[2].name, data.users[2].nb, (data.users[2].nb*100)/data.totalMessagesPublic, data.users[2].reply)
		}

		m = m + "### :see_no_evil: Podium Channels Conversations\n"
		if len(data.channels) > 0 {
			m = m + fmt.Sprintf("* :1st_place_medal: ~%s with a total of **%d** *(%d%%)* messages with %d reply\n", data.channels[0].name, data.channels[0].nb, (data.channels[0].nb*100)/(data.totalMessagesPublic+data.totalMessagesPrivate), data.channels[0].reply)
		}
		if len(data.channels) > 1 {
			m = m + fmt.Sprintf("* :2nd_place_medal: ~%s with a total of **%d** *(%d%%)* messages with %d reply\n", data.channels[1].name, data.channels[1].nb, (data.channels[1].nb*100)/(data.totalMessagesPublic+data.totalMessagesPrivate), data.channels[1].reply)
		}
		if len(data.channels) > 2 {
			m = m + fmt.Sprintf("* :3rd_place_medal: ~%s with a total of **%d** *(%d%%)* messages with %d reply\n", data.channels[2].name, data.channels[2].nb, (data.channels[2].nb*100)/(data.totalMessagesPublic+data.totalMessagesPrivate), data.channels[2].reply)
		}
	}

	urlPie, _ := url.Parse("http://127.0.0.1:8065/plugins/com.github.manland.mattermost-plugin-analytics/pie.svg")
	parametersURLPie := url.Values{}
	for _, c := range data.channels {
		parametersURLPie.Add(c.displayName, fmt.Sprintf("%d", c.nb))
	}
	urlPie.RawQuery = parametersURLPie.Encode()
	pie := "![](" + urlPie.String() + ") "

	urlBar, _ := url.Parse("http://127.0.0.1:8065/plugins/com.github.manland.mattermost-plugin-analytics/bar.svg")
	parametersURLBar := url.Values{}
	for _, c := range data.users {
		parametersURLBar.Add(c.displayName, fmt.Sprintf("%d", c.nb))
	}
	urlBar.RawQuery = parametersURLBar.Encode()
	bar := "![](" + urlBar.String() + ") "

	allSessions, _ := p.allSessions()
	urlLine, _ := url.Parse("http://127.0.0.1:8065/plugins/com.github.manland.mattermost-plugin-analytics/line.svg")
	parametersURLLine := url.Values{}
	allChannels := make(map[string]bool, 0)
	for _, session := range allSessions {
		for key := range session.Channels {
			allChannels[key] = true
		}
	}
	for _, session := range allSessions {
		for key := range allChannels {
			allChannels[key] = false //init with not found
		}
		for key, value := range session.Channels {
			displayKey, err := p.getChannelDisplayName(key)
			if err != nil {
				return nil, &model.AppError{Message: err.Error()}
			}
			allChannels[key] = true
			parametersURLLine.Add(displayKey, fmt.Sprintf("%d", value))
		}
		for key := range allChannels {
			displayKey, err := p.getChannelDisplayName(key)
			if err != nil {
				return nil, &model.AppError{Message: err.Error()}
			}
			if !allChannels[key] {
				parametersURLLine.Add(displayKey, "0")
			}
		}
		parametersURLLine.Add("date", fmt.Sprintf("%d", session.Start.Unix()))
	}
	urlLine.RawQuery = parametersURLLine.Encode()
	line := "![](" + urlLine.String() + ") "

	m = m + pie + bar + line

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Text:         m,
	}, nil

}

type analyticsData struct {
	displayName string
	name        string
	nb          int64
	reply       int64
}

type preparedData struct {
	totalMessagesPublic  int64
	totalMessagesPrivate int64
	users                []analyticsData
	channels             []analyticsData
}

func (p *Plugin) prepareData() (*preparedData, error) {
	p.currentAnalytic.RLock()
	defer p.currentAnalytic.RUnlock()

	totalMessagesPublic := int64(0)
	totalMessagesPrivate := int64(0)
	users := make([]analyticsData, 0)
	channels := make([]analyticsData, 0)
	channels = append(channels, analyticsData{"Private", "Private", 0, 0})

	for key, nb := range p.currentAnalytic.Channels {
		channelName, channelDisplayName, err := p.getChannelName(key)
		if err != nil {
			return nil, err
		}
		if channelName == "Private" {
			totalMessagesPrivate += nb
			channels[0].nb = channels[0].nb + nb
		} else {
			totalMessagesPublic += nb
			channels = p.updateOrAppend(channels, analyticsData{channelDisplayName, channelName, nb, 0})
		}
	}
	for key, nb := range p.currentAnalytic.ChannelsReply {
		channelName, channelDisplayName, err := p.getChannelName(key)
		if err != nil {
			return nil, err
		}
		channels = p.updateOrAppend(channels, analyticsData{channelDisplayName, channelName, 0, nb})
	}
	for key, nb := range p.currentAnalytic.Users {
		displayKey, err := p.getUsername(key)
		if err != nil {
			return nil, err
		}
		users = p.updateOrAppend(users, analyticsData{displayKey, displayKey, nb, 0})
	}
	for key, nb := range p.currentAnalytic.UsersReply {
		displayKey, err := p.getUsername(key)
		if err != nil {
			return nil, err
		}
		users = p.updateOrAppend(users, analyticsData{displayKey, displayKey, 0, nb})
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].nb > users[j].nb
	})
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].nb > channels[j].nb
	})
	return &preparedData{
		totalMessagesPublic:  totalMessagesPublic,
		totalMessagesPrivate: totalMessagesPrivate,
		users:                users,
		channels:             channels,
	}, nil
}

func (p *Plugin) updateOrAppend(originals []analyticsData, upsert analyticsData) []analyticsData {
	var original analyticsData
	for index, value := range originals {
		if value.name == upsert.name {
			original = value
			originals = append(originals[:index], originals[index+1:]...)
			break
		}
	}
	if original.name != "" {
		if upsert.nb == 0 {
			upsert.nb = original.nb
		}
		if upsert.reply == 0 {
			upsert.reply = original.reply
		}
	}
	return append(originals, upsert)
}

func (p *Plugin) getChannelName(key string) (string, string, error) {
	channel, err := p.API.GetChannel(key)
	if err != nil {
		return "", "", errors.Wrap(err, "Can't retreive channel name")
	}
	if channel.IsGroupOrDirect() {
		return "", "Private", nil
	}
	return channel.Name, channel.DisplayName, nil
}

func (p *Plugin) getChannelDisplayName(key string) (string, error) {
	channel, err := p.API.GetChannel(key)
	if err != nil {
		return "", errors.Wrap(err, "Can't retreive channel display name")
	}
	if channel.IsGroupOrDirect() {
		return "Private", nil
	}
	return channel.DisplayName, nil
}

func (p *Plugin) getUsername(key string) (string, error) {
	user, err := p.API.GetUser(key)
	if err != nil {
		return "", errors.Wrap(err, "Can't retreive user name")
	}
	return user.Username, nil
}

func byteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
