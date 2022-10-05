package comment

import (
	"time"

	"crud/pkg/user"
)

type CommentId string

type Comment struct {
	Id      CommentId  `json:"id"`
	Author  *user.User `json:"author"`
	Created time.Time  `json:"created"`
	Body    string     `json:"body"`
}
