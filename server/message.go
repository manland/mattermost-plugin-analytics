package main

import (
	"io"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	p.currentAnalytic.WLock()
	defer p.currentAnalytic.WUnlock()

	p.currentAnalytic.Users[post.UserId]++
	p.currentAnalytic.Channels[post.ChannelId]++
	if post.ParentId != "" {
		p.currentAnalytic.UsersReply[post.UserId]++
		p.currentAnalytic.ChannelsReply[post.ChannelId]++
	}
}

func (p *Plugin) FileWillBeUploaded(c *plugin.Context, info *model.FileInfo, file io.Reader, output io.Writer) (*model.FileInfo, string) {
	p.currentAnalytic.WLock()
	defer p.currentAnalytic.WUnlock()
	p.currentAnalytic.FilesNb++
	p.currentAnalytic.FilesSize += info.Size
	return info, ""
}
