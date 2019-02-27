package main

import (
	"fmt"
	"net/url"

	"github.com/mattermost/mattermost-server/model"
)

const (
	maxChannelsToDisplay = 10
	maxUsersToDisplay    = 10
)

func (p *Plugin) buildAnalyticMsg() (string, error) {
	data, err := p.prepareData()
	if err != nil {
		return "", err
	}

	p.currentAnalytic.RLock()
	m := fmt.Sprintf("## Analytics since %s at %s\n", p.currentAnalytic.Start.Format("2 January"), p.currentAnalytic.Start.Format("15:04"))
	p.currentAnalytic.RUnlock()
	if data.totalMessagesPublic+data.totalMessagesPrivate > 0 {
		m = m + fmt.Sprintf("#### **%d users** sent a total of **%d messages** in **%d channels**. With **%d** *(%d%%)* in public channels and **%d** *(%d%%)* in private.\n", len(data.users), data.totalMessagesPublic+data.totalMessagesPrivate, len(data.channels), data.totalMessagesPublic, (data.totalMessagesPublic*100)/(data.totalMessagesPublic+data.totalMessagesPrivate), data.totalMessagesPrivate, (data.totalMessagesPrivate*100)/(data.totalMessagesPublic+data.totalMessagesPrivate))
		m = m + fmt.Sprintf("#### And they sent a total of **%d files** for a total of **%s**.\n", p.currentAnalytic.FilesNb, byteCountDecimal(p.currentAnalytic.FilesSize))

		m = m + "### :speak_no_evil: Podium Speaker Users\n"
		if len(data.users) > 0 {
			m = m + fmt.Sprintf("* :1st_place_medal: @%s with a total of **%d** *(%d%%)* public messages with %d reply\n", data.users[0].name, data.users[0].nb, getPercentComparingToPublicMessages(data, data.users[0]), data.users[0].reply)
		}
		if len(data.users) > 1 {
			m = m + fmt.Sprintf("* :2nd_place_medal: @%s with a total of **%d** *(%d%%)* public messages with %d reply\n", data.users[1].name, data.users[1].nb, getPercentComparingToPublicMessages(data, data.users[1]), data.users[1].reply)
		}
		if len(data.users) > 2 {
			m = m + fmt.Sprintf("* :3rd_place_medal: @%s with a total of **%d** *(%d%%)* public messages with %d reply\n", data.users[2].name, data.users[2].nb, getPercentComparingToPublicMessages(data, data.users[2]), data.users[2].reply)
		}

		m = m + "### :see_no_evil: Podium Channels Conversations\n"
		if len(data.channels) > 0 {
			m = m + fmt.Sprintf("* :1st_place_medal: %s with a total of **%d** *(%d%%)* messages with %d reply\n", getChannelLink(data.channels[0]), data.channels[0].nb, getPercentComparingToAllMessages(data, data.channels[0]), data.channels[0].reply)
		}
		if len(data.channels) > 1 {
			m = m + fmt.Sprintf("* :2nd_place_medal: %s with a total of **%d** *(%d%%)* messages with %d reply\n", getChannelLink(data.channels[1]), data.channels[1].nb, getPercentComparingToAllMessages(data, data.channels[1]), data.channels[1].reply)
		}
		if len(data.channels) > 2 {
			m = m + fmt.Sprintf("* :3rd_place_medal: %s with a total of **%d** *(%d%%)* messages with %d reply\n", getChannelLink(data.channels[2]), data.channels[2].nb, getPercentComparingToAllMessages(data, data.channels[2]), data.channels[2].reply)
		}
	}

	urlPie, _ := url.Parse("http://127.0.0.1:8065/plugins/com.github.manland.mattermost-plugin-analytics/pie.svg")
	parametersURLPie := url.Values{}
	for index, c := range data.channels {
		if index > maxChannelsToDisplay {
			break
		}
		parametersURLPie.Add(c.displayName, fmt.Sprintf("%d", c.nb))
	}
	urlPie.RawQuery = parametersURLPie.Encode()
	pie := fmt.Sprintf("![channels pie chart](%s) ", urlPie.String())

	urlBar, _ := url.Parse("http://127.0.0.1:8065/plugins/com.github.manland.mattermost-plugin-analytics/bar.svg")
	parametersURLBar := url.Values{}
	for index, c := range data.users {
		if index > maxUsersToDisplay {
			break
		}
		parametersURLBar.Add(c.displayName, fmt.Sprintf("%d", c.nb))
	}
	urlBar.RawQuery = parametersURLBar.Encode()
	bar := fmt.Sprintf("![users bar chart](%s) ", urlBar.String())

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
				return "", err
			}
			allChannels[key] = true
			parametersURLLine.Add(displayKey, fmt.Sprintf("%d", value))
		}
		for key := range allChannels {
			displayKey, err := p.getChannelDisplayName(key)
			if err != nil {
				return "", err
			}
			if !allChannels[key] {
				parametersURLLine.Add(displayKey, "0")
			}
		}
		parametersURLLine.Add("date", fmt.Sprintf("%d", session.Start.Unix()))
	}
	urlLine.RawQuery = parametersURLLine.Encode()
	line := fmt.Sprintf("![all sessions line chart](%s) ", urlLine.String())

	return m + pie + bar + line, nil
}

func (p *Plugin) sendAnalytics() error {
	m, err := p.buildAnalyticMsg()
	if err != nil {
		return err
	}
	for _, channelID := range p.ChannelsID {
		post := &model.Post{
			UserId:    p.BotUserID,
			ChannelId: channelID,
			Message:   m,
			Props: map[string]interface{}{
				"from_webhook":      "true",
				"override_username": p.getConfiguration().BotUsername,
				"override_icon_url": p.getConfiguration().BotIconURL,
			},
		}

		if _, err := p.API.CreatePost(post); err != nil {
			return err
		}
	}

	return nil
}

func getChannelLink(data analyticsData) string {
	if data.displayName != dmOrPrivateChannelName {
		return fmt.Sprintf("[~%s](%s)", data.displayName, data.link)
	}
	return data.displayName
}

func getPercentComparingToPublicMessages(prepared *preparedData, data analyticsData) int64 {
	return (data.nb * 100) / prepared.totalMessagesPublic
}

func getPercentComparingToAllMessages(prepared *preparedData, data analyticsData) int64 {
	return (data.nb * 100) / (prepared.totalMessagesPublic + prepared.totalMessagesPrivate)
}
