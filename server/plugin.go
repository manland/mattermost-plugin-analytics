package main

import (
	"fmt"
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

	cron *Cron

	BotUserID  string
	ChannelsID []string
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

	m, err := p.buildAnalyticMsg()
	if err != nil {
		return nil, &model.AppError{Message: err.Error()}
	}

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
		return "Private", "Private", nil
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
