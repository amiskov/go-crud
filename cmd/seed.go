package main

import (
	"context"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/jaswdr/faker"

	"crud/pkg/comment"
	. "crud/pkg/common"
	"crud/pkg/post"
	"crud/pkg/user"
)

const (
	typeText = "text"
	typeLink = "link"
)

var (
	f             = faker.New()
	onePassForAll = HashPass("sdfsdfsdf", RandStringRunes(8)) // salt must have len of 8
)

type IUserRepo interface {
	Add(*user.User) (string, error)
	GetAll() ([]*user.User, error)
}

func createAuthors(userRepo IUserRepo) {
	// User for experiments (not random)
	_, err := userRepo.Add(&user.User{
		Username: "pike",
		Password: onePassForAll,
	})
	if err != nil {
		log.Fatalln("seed: can't create default user:", err)
	}
	for i := 1; i <= 5; i++ {
		genUser(userRepo, i)
	}
}

func seed(userRepo IUserRepo, postRepo *post.Repo) {
	authors, err := userRepo.GetAll()
	if err != nil {
		log.Fatalln("seed: can't get all authors:", err)
	}

	if len(authors) == 0 {
		createAuthors(userRepo)
	}

	for i := 0; i <= 5; i++ {
		_, err := postRepo.Add(context.Background(), genPost(authors))
		if err != nil {
			log.Fatalln("seed: can't add post:", err)
		}
	}
}

func randCategory() string {
	categories := []string{"programming", "music", "videos", "funny", "news", "fashion"}
	n := rand.Int() % len(categories)
	return categories[n]
}

func randType() string {
	types := []string{typeText, typeLink}
	return types[rand.Intn(2)]
}

func genUser(userRepo IUserRepo, id int) {
	username := strings.ToLower(f.Person().FirstName())
	u := user.User{
		// ID is made from its because we want them the same after server reloading
		Id:       strconv.Itoa(id),
		Username: username,
		Password: onePassForAll,
	}
	_, err := userRepo.Add(&u)
	if err != nil {
		log.Fatalln("seed: can't add user:", err)
	}
}

func genComments(users []*user.User) []*comment.Comment {
	n := rand.Intn(10)
	comments := []*comment.Comment{}
	author := randUser(users)
	for i := 0; i <= n; i++ {
		comments = append(comments, &comment.Comment{
			Id:      comment.CommentId(RandStringRunes(12)),
			Author:  author,
			Created: f.Time().Time(time.Now()),
			Body:    genText(),
		})
	}
	return comments
}

func genTitle() string {
	return strings.Join(f.Lorem().Words(rand.Intn(5)+3), " ")
}

func genText() string {
	return f.Lorem().Paragraph(rand.Intn(3) + 2)
}

func genPost(users []*user.User) *post.Post {
	postType := randType()
	var postText string
	var postURL string
	if postType == typeLink {
		postURL = f.Address().Faker.Internet().URL()
	} else if postType == typeText {
		postText = genText()
	}

	return &post.Post{
		Author:   randUser(users),
		Id:       post.PostId(RandStringRunes(12)),
		Title:    genTitle(),
		Type:     postType,
		Text:     postText,
		URL:      postURL,
		Category: randCategory(),
		Views:    rand.Intn(100),
		Created:  f.Time().Time(time.Now()),
		Comments: genComments(users),
	}
}

func randUser(users []*user.User) *user.User {
	idx := rand.Intn(len(users))
	return users[idx]
}
