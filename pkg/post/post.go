package post

import (
	"time"

	"crud/pkg/comment"
	"crud/pkg/user"
	"crud/pkg/voting"
)

const (
	PostText = "text"
	PostLink = "link"
)

type PostId string

type Post struct {
	Author   *user.User         `json:"author"`
	Comments []*comment.Comment `json:"comments"`
	Votes    []*voting.Vote     `json:"votes"`
	Id       PostId             `json:"id"`
	Title    string             `json:"title"`

	// Types: [text|link].
	Type string `json:"type"`

	// Text for type "text", URL for type "link"
	Text string `json:"text"`
	URL  string `json:"url"`

	Category         string    `json:"category"`
	Views            int       `json:"views"`
	Score            int       `json:"score"`
	UpvotePercentage int       `json:"upvotePercentage"`
	Created          time.Time `json:"created"`
}
