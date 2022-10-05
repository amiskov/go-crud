package post

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"crud/pkg/comment"
	. "crud/pkg/common"
	"crud/pkg/logger"
	"crud/pkg/sessions"
	"crud/pkg/user"
	"crud/pkg/voting"
)

type IPostRepo interface {
	GetAll(context.Context) ([]*Post, error)
	GetById(context.Context, PostId) (*Post, error)
	GetCategoryPosts(context.Context, string) ([]*Post, error)
	GetUserPosts(context.Context, string) ([]*Post, error)

	Add(context.Context, *Post) (PostId, error)
	Update(context.Context, *Post) error

	Vote(context.Context, *Post, *voting.Vote) error

	Delete(context.Context, PostId) error
	DeletePostComment(context.Context, PostId, comment.CommentId) (*Post, error)

	AddComment(context.Context, PostId, *user.User, string) (*Post, error)
}

type PostHandler struct {
	PostRepo IPostRepo
}

func NewPostHandler(postRepo IPostRepo) *PostHandler {
	return &PostHandler{
		PostRepo: postRepo,
	}
}

func (ph PostHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := ph.PostRepo.GetAll(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("can't load posts from the repo: %v", err)
		WriteMsg(w, "failed loading posts", http.StatusInternalServerError)
		return
	}

	WriteRespJSON(w, posts)
}

func (ph *PostHandler) Add(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	author, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("post/handlers: can't get the user form repo: %v", err)
		WriteMsg(w, "user not found", http.StatusBadRequest)
		return
	}

	post := new(Post)
	err = ParseReqBody(r.Body, post)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse post from request body: %v", err)
		WriteMsg(w, "can't parse post", http.StatusBadRequest)
		return
	}

	post.Created = time.Now()
	post.Id = PostId(RandStringRunes(12))
	post.Author = author
	post.Votes = make([]*voting.Vote, 0)
	post.Comments = make([]*comment.Comment, 0)

	_, err = ph.PostRepo.Add(r.Context(), post)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't add post to the repo: %v", err)
		WriteMsg(w, "failed adding post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	WriteRespJSON(w, post)
}

func (ph *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	postId := vars["post_id"]

	authUser, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("can't find auth user: %v", err)
		WriteMsg(w, "user not found", http.StatusBadRequest)
		return
	}

	post, err := ph.PostRepo.GetById(r.Context(), PostId(postId))
	if err != nil {
		logger.Log(r.Context()).Errorf("can't find the post: %v", err)
		WriteMsg(w, "post not found", http.StatusInternalServerError)
		return
	}

	if post.Author.Id != authUser.Id {
		logger.Log(r.Context()).Errorf("post Id is not the same as the auth user Id: %v", err)
		WriteMsg(w, "only the author can remove the post", http.StatusInternalServerError)
		return
	}

	err = ph.PostRepo.Delete(r.Context(), PostId(postId))
	if err != nil {
		logger.Log(r.Context()).Errorf("can't remove post: %v", err)
		WriteMsg(w, "removing post failed", http.StatusInternalServerError)
		return
	}

	WriteMsg(w, "success", http.StatusOK)
}

func (ph *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	postId := PostId(vars["post_id"])
	commentId := comment.CommentId(vars["comment_id"])

	postWithoutComment, err := ph.PostRepo.DeletePostComment(r.Context(), postId, commentId)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't remove comment %s from post %s: %v", commentId, postId, err)
		WriteMsg(w, "removing comment failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	WriteRespJSON(w, postWithoutComment)
}

func (ph *PostHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	postId := vars["post_id"]

	c := struct{ Comment string }{}
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get comment body: %v", err)
		WriteMsg(w, "failed parsing comment body", http.StatusInternalServerError)
		return
	}

	commenter, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("can't find the user by username: %v", err)
		WriteMsg(w, "not authorized", http.StatusInternalServerError)
		return
	}

	postWithComment, err := ph.PostRepo.AddComment(r.Context(), PostId(postId), commenter, c.Comment)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't add comment %s: %v", postId, err)
		WriteMsg(w, "post not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusCreated)
	WriteRespJSON(w, postWithComment)
}

func (ph *PostHandler) Upvote(w http.ResponseWriter, r *http.Request) {
	ph.vote(w, r, voting.ScoreUp)
}

func (ph *PostHandler) Unvote(w http.ResponseWriter, r *http.Request) {
	ph.vote(w, r, voting.ScoreDiscard)
}

func (ph *PostHandler) Downvote(w http.ResponseWriter, r *http.Request) {
	ph.vote(w, r, voting.ScoreDown)
}

func (ph *PostHandler) vote(w http.ResponseWriter, r *http.Request, score voting.VotingScore) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	postId := vars["post_id"]

	voter, err := sessions.GetAuthUser(r.Context())
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get user from JWT token: %v", err)
		WriteMsg(w, "not authorized", http.StatusBadRequest)
		return
	}

	post, err := ph.PostRepo.GetById(r.Context(), PostId(postId))
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get post with id %s: %v", postId, err)
		WriteMsg(w, "post not found", http.StatusNotFound)
		return
	}

	v := &voting.Vote{
		UserId: voter.Id,
		Score:  score,
	}

	err = ph.PostRepo.Vote(r.Context(), post, v)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't vote for post %s: %v", postId, err)
		WriteMsg(w, "voting failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	WriteRespJSON(w, post)
}

func (ph PostHandler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	postId := vars["post_id"]
	post, err := ph.PostRepo.GetById(r.Context(), PostId(postId))
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get post with id %s: %v", postId, err)
		WriteMsg(w, "post not found", http.StatusNotFound)
		return
	}

	WriteRespJSON(w, post)
}

func (ph PostHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	category := vars["category"]

	categoryPosts, err := ph.PostRepo.GetCategoryPosts(r.Context(), category)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't load category %s %s: %v", category, err)
		WriteMsg(w, "failed loading posts for the category", http.StatusInternalServerError)
		return
	}

	WriteRespJSON(w, categoryPosts)
}

func (ph PostHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	username := vars["username"]

	userPosts, err := ph.PostRepo.GetUserPosts(r.Context(), username)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't load user `%s` posts from the repo %s %s: %v", username, err)
		WriteMsg(w, "failed loading user posts", http.StatusInternalServerError)
		return
	}

	WriteRespJSON(w, userPosts)
}
