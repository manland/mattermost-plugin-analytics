package main

import (
	"io"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// MessageHasBeenPosted is called by mattermost when a message has been posted
// used to store metrics on messages
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

// FileWillBeUploaded is called by mattermost when a file will be uploaded
// used to store number of files and weight
func (p *Plugin) FileWillBeUploaded(c *plugin.Context, info *model.FileInfo, file io.Reader, output io.Writer) (*model.FileInfo, string) {
	p.currentAnalytic.WLock()
	defer p.currentAnalytic.WUnlock()

	p.currentAnalytic.FilesNb++
	p.currentAnalytic.FilesSize += info.Size
	return info, ""
}
