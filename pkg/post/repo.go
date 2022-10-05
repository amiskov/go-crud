package post

import (
	"context"
	"fmt"
	"time"

	"crud/pkg/comment"
	"crud/pkg/common"
	"crud/pkg/logger"
	"crud/pkg/user"
	"crud/pkg/voting"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repo struct {
	posts IMongoCollection
}

func NewPostRepo(postsCol *mongo.Collection) *Repo {
	posts := &MongoCollection{
		Coll: postsCol,
	}
	return &Repo{
		posts: posts,
	}
}

func (r *Repo) Add(ctx context.Context, p *Post) (PostId, error) {
	_, err := r.posts.InsertOne(ctx, p)
	if err != nil {
		return PostId(``), fmt.Errorf("post/repo: failed inserting a post: %w", err)
	}
	return PostId(p.Id), nil
}

func (r *Repo) Update(ctx context.Context, p *Post) error {
	_, err := r.posts.UpdateOne(ctx, bson.M{"id": p.Id}, bson.M{"$set": p})
	if err != nil {
		return fmt.Errorf("post/repo: failed updating post: %w", err)
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id PostId) error {
	_, err := r.posts.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("post/repo: failed deleting post: %w", err)
	}
	return nil
}

func (r *Repo) GetById(ctx context.Context, id PostId) (*Post, error) {
	post := new(Post)
	err := r.posts.FindOne(ctx, bson.M{"id": id}).Decode(post)
	if err != nil {
		return nil, fmt.Errorf("post: post not found: %w", err)
	}
	return post, nil
}

func (r *Repo) GetAll(ctx context.Context) ([]*Post, error) {
	cursor, err := r.posts.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("post/repo: failed finding posts: %w", err)
	}
	defer cursor.Close(ctx)

	posts := []*Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, fmt.Errorf("post/repo: failed geting posts from cursor: %w", err)
	}
	return posts, nil
}

func (r *Repo) AddComment(ctx context.Context, postId PostId, commenter *user.User, commentText string) (*Post, error) {
	cmt := new(comment.Comment)
	cmt.Id = comment.CommentId(common.RandStringRunes(12))
	cmt.Author = commenter
	cmt.Created = time.Now()
	cmt.Body = commentText

	filter := bson.D{{Key: "id", Value: postId}}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "comments", Value: cmt}}}}
	_, err := r.posts.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	post, err := r.GetById(ctx, PostId(postId))
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (r *Repo) DeletePostComment(ctx context.Context, postId PostId, commentId comment.CommentId) (*Post, error) {
	filter := bson.D{{"id", postId}}
	update := bson.D{{"$pull", bson.D{{"comments", bson.D{{"id", commentId}}}}}}
	_, err := r.posts.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	post, err := r.GetById(ctx, postId)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (r *Repo) GetCategoryPosts(ctx context.Context, category string) ([]*Post, error) {
	cursor, err := r.posts.Find(ctx, bson.D{{"category", category}})
	if err != nil {
		return nil, fmt.Errorf("post/repo: failed finding posts: %w", err)
	}

	categoryPosts := []*Post{}
	if err := cursor.All(ctx, &categoryPosts); err != nil {
		return nil, fmt.Errorf("post/repo: failed geting posts from cursor: %w", err)
	}

	return categoryPosts, nil
}

func (r *Repo) GetUserPosts(ctx context.Context, username string) ([]*Post, error) {
	userPosts := []*Post{}
	filter := bson.D{{Key: "author.username", Value: username}}
	cursor, err := r.posts.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("post/repo: failed finding posts: %w", err)
	}
	if err := cursor.All(ctx, &userPosts); err != nil {
		return nil, fmt.Errorf("post/repo: failed geting posts from cursor: %w", err)
	}
	return userPosts, nil
}

// Vote with the MongoDB transaction:
// - update the post data,
// - recalculate upvote percentage
// - and update the post in DB.
func (r *Repo) Vote(ctx context.Context, post *Post, newVote *voting.Vote) error {
	session, err := r.posts.Database().Client().StartSession()
	if err != nil {
		logger.Log(ctx).Errorf("post/repo: start session failed", err)
		return err
	}
	defer session.EndSession(ctx)

	callback := func(sessionContext mongo.SessionContext) (interface{}, error) {
		_, userAlreadyVoted := r.getUserVoteFromPost(post, newVote.UserId)

		if !userAlreadyVoted && newVote.Score != voting.ScoreDiscard {
			post.Votes = append(post.Votes, newVote)
		} else {
			// This handles several cases which are equal to unvote:
			// 1. User wants remove previous vote (pure unvote);
			// 2. User previously voted +1 and now votes -1;
			// 3. User previously voted -1 and now votes +1.
			removeVote(post, newVote.UserId)
		}

		updatePercentage(post)

		err := r.Update(ctx, post)
		if err != nil {
			return nil, fmt.Errorf("post/repo: can't update vote %w", err)
		}

		return post, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		logger.Log(ctx).Errorf("post/repo: failed update post votes", err)
		return err
	}

	return nil
}

func removeVote(post *Post, userId string) {
	for idx, v := range post.Votes {
		if v.UserId == userId {
			// remove from slice
			post.Votes[idx] = post.Votes[len(post.Votes)-1]
			post.Votes[len(post.Votes)-1] = &voting.Vote{}
			post.Votes = post.Votes[:len(post.Votes)-1]
			break
		}
	}
}

func (r *Repo) getUserVoteFromPost(post *Post, userId string) (*voting.Vote, bool) {
	for _, v := range post.Votes {
		if v.UserId == userId {
			return v, true
		}
	}
	return nil, false
}

// Recalculate upvote percentage of the post
func updatePercentage(post *Post) {
	var upCount, downCount int
	for _, v := range post.Votes {
		if v.Score == 1 {
			upCount++
		} else if v.Score == -1 {
			downCount++
		}
	}
	score := upCount - downCount
	totalVotes := upCount + downCount
	if totalVotes > 0 {
		post.UpvotePercentage = int((upCount / totalVotes) * 100)
	} else {
		post.UpvotePercentage = 0
	}
	post.Score = score
}
