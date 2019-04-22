package main

import (
	"fmt"
	"net/url"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

const (
	maxChannelsToDisplay = 10
	maxUsersToDisplay    = 10
)

func (p *Plugin) buildAnalyticAttachments() ([]*model.SlackAttachment, error) {
	siteURL := p.API.GetConfig().ServiceSettings.SiteURL

	data, err := p.prepareData()
	if err != nil {
		return nil, err
	}

	p.currentAnalytic.RLock()
	text := fmt.Sprintf("## Analytics since %s, at %s.\n", p.currentAnalytic.Start.Format("January 2, 2006"), p.currentAnalytic.Start.Format("15:04"))
	p.currentAnalytic.RUnlock()
	if data.totalMessagesPublic+data.totalMessagesPrivate > 0 {
		text += fmt.Sprintf("#### **%d users** sent **%d messages** in **%d channels**. **%d** *(%d%%)* of the messages were in public channels, **%d** *(%d%%)* in private.\n", len(data.users), data.totalMessagesPublic+data.totalMessagesPrivate, len(data.channels), data.totalMessagesPublic, (data.totalMessagesPublic*100)/(data.totalMessagesPublic+data.totalMessagesPrivate), data.totalMessagesPrivate, (data.totalMessagesPrivate*100)/(data.totalMessagesPublic+data.totalMessagesPrivate))
		text += fmt.Sprintf("#### Moreover, **%d files** were sent for a total uppload size of **%s**.\n", p.currentAnalytic.FilesNb, byteCountDecimal(p.currentAnalytic.FilesSize))
	}

	fields := append(getUsersFields(*siteURL, data), getChannelsFields(*siteURL, data)...)
	sessions, err := p.getSessionsFields(*siteURL)
	if err != nil {
		return nil, err
	}
	fields = append(fields, sessions...)

	attachments := make([]*model.SlackAttachment, 1)
	attachments[0] = &model.SlackAttachment{
		Color:  "#FF8000",
		Text:   text,
		Fields: fields,
	}

	return attachments, nil
}

func (p *Plugin) sendAnalytics(ChannelsID []string) error {
	attachments, err := p.buildAnalyticAttachments()
	if err != nil {
		return errors.Wrap(err, "can't build analytics attachments")
	}
	for _, channelID := range ChannelsID {
		post := &model.Post{
			UserId:    p.BotUserID,
			ChannelId: channelID,
			Props: map[string]interface{}{
				"from_webhook":      "true",
				"override_username": p.getConfiguration().BotUsername,
				"override_icon_url": p.getConfiguration().BotIconURL,
				"attachments":       attachments,
			},
		}

		if _, err := p.API.CreatePost(post); err != nil {
			return errors.Wrap(err, "can't post mesage")
		}
	}

	return nil
}

func getUsersFields(siteURL string, data *preparedData) []*model.SlackAttachmentField {
	m := "### Top Users\n"
	if len(data.users) > 0 {
		m = m + fmt.Sprintf("* :1st_place_medal: @%s: **%d** messages *(%d%% of total)* with %d replies.\n", data.users[0].name, data.users[0].nb, getPercentComparingToPublicMessages(data, data.users[0]), data.users[0].reply)
	}
	if len(data.users) > 1 {
		m = m + fmt.Sprintf("* :2nd_place_medal: @%s: **%d** messages *(%d%% of total)* with %d replies.\n", data.users[1].name, data.users[1].nb, getPercentComparingToPublicMessages(data, data.users[1]), data.users[1].reply)
	}
	if len(data.users) > 2 {
		m = m + fmt.Sprintf("* :3rd_place_medal: @%s: **%d** messages *(%d%% of total)* with %d replies.\n", data.users[2].name, data.users[2].nb, getPercentComparingToPublicMessages(data, data.users[2]), data.users[2].reply)
	}
	urlChart, _ := url.Parse(siteURL + "/plugins/com.github.manland.mattermost-plugin-analytics/pie.svg")
	parametersURL := url.Values{}
	for index, c := range data.users {
		if index > maxUsersToDisplay {
			break
		}
		parametersURL.Add(c.displayName, fmt.Sprintf("%d", c.nb))
	}
	urlChart.RawQuery = parametersURL.Encode()
	return buildSlackAttachmentField(m, "users pie chart", urlChart)
}

func getChannelsFields(siteURL string, data *preparedData) []*model.SlackAttachmentField {
	m := "### Top Channels\n"
	if len(data.channels) > 0 {
		m = m + fmt.Sprintf("* :1st_place_medal: %s: **%d** messages *(%d%% of total)* with %d replies.\n", getChannelLink(data.channels[0]), data.channels[0].nb, getPercentComparingToAllMessages(data, data.channels[0]), data.channels[0].reply)
	}
	if len(data.channels) > 1 {
		m = m + fmt.Sprintf("* :2nd_place_medal: %s: **%d** messages *(%d%% of total)* with %d replies.\n", getChannelLink(data.channels[1]), data.channels[1].nb, getPercentComparingToAllMessages(data, data.channels[1]), data.channels[1].reply)
	}
	if len(data.channels) > 2 {
		m = m + fmt.Sprintf("* :3rd_place_medal: %s: **%d** messages *(%d%% of total)* with %d replies.\n", getChannelLink(data.channels[2]), data.channels[2].nb, getPercentComparingToAllMessages(data, data.channels[2]), data.channels[2].reply)
	}
	urlChart, _ := url.Parse(siteURL + "/plugins/com.github.manland.mattermost-plugin-analytics/pie.svg")
	parametersURL := url.Values{}
	for index, c := range data.channels {
		if index > maxChannelsToDisplay {
			break
		}
		parametersURL.Add(c.displayName, fmt.Sprintf("%d", c.nb))
	}
	urlChart.RawQuery = parametersURL.Encode()
	return buildSlackAttachmentField(m, "channels pie chart", urlChart)
}

func (p *Plugin) getSessionsFields(siteURL string) ([]*model.SlackAttachmentField, error) {
	allSessions, _ := p.allSessions()
	urlChart, _ := url.Parse(siteURL + "/plugins/com.github.manland.mattermost-plugin-analytics/line.svg")
	parametersURL := url.Values{}
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
				return nil, err
			}
			allChannels[key] = true
			parametersURL.Add(displayKey, fmt.Sprintf("%d", value))
		}
		for key := range allChannels {
			displayKey, err := p.getChannelDisplayName(key)
			if err != nil {
				return nil, err
			}
			if !allChannels[key] {
				parametersURL.Add(displayKey, "0")
			}
		}
		parametersURL.Add("date", fmt.Sprintf("%d", session.Start.Unix()))
	}
	urlChart.RawQuery = parametersURL.Encode()
	return buildSlackAttachmentField("", "all sessions line chart", urlChart), nil
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

func buildSlackAttachmentField(description string, chartTitle string, chartURL *url.URL) []*model.SlackAttachmentField {
	attachments := make([]*model.SlackAttachmentField, 0)
	if description != "" {
		attachments = append(attachments, &model.SlackAttachmentField{Short: true, Value: description})
	}
	return append(attachments, &model.SlackAttachmentField{
		Short: true,
		// make a md array to have little border around image, working with all themes
		Value: fmt.Sprintf("| |\n|:-:|\n|![%s](%s)|", chartTitle, chartURL.String()),
	},
	)
}
