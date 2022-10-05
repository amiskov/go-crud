package post

import (
	"context"
	"crud/pkg/user"
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPostAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockMongoColl := NewMockIMongoCollection(ctrl)
	mockInserOneResult := NewMockIMongoInsertOneResult(ctrl)

	repo := &Repo{
		posts: mockMongoColl,
	}

	testPost := &Post{Id: PostId("1")}

	t.Run("success", func(t *testing.T) {
		mockMongoColl.EXPECT().
			InsertOne(ctx, gomock.Any()).
			Return(mockInserOneResult, nil)

		insertedPostId, err := repo.Add(context.Background(), testPost)
		if err != nil {
			t.Errorf("failed success test %v", err)
			return
		}
		assert.Nil(t, err)
		assert.Equal(t, testPost.Id, insertedPostId)
	})

	t.Run("insert error", func(t *testing.T) {
		expectedErr := fmt.Errorf("insert_failed")
		mockMongoColl.EXPECT().
			InsertOne(ctx, gomock.Any()).
			Return(nil, expectedErr)

		insertedPostId, err := repo.Add(context.Background(), &Post{})
		assert.Equal(t, insertedPostId, PostId(``))
		assert.NotNil(t, err)
	})
}

func TestGetUserPosts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockMongoColl := NewMockIMongoCollection(ctrl)
	mockFindResult := NewMockIMongoCursor(ctrl)

	repo := &Repo{
		posts: mockMongoColl,
	}

	t.Run("success", func(t *testing.T) {
		username := "pike"
		expectedPosts := []*Post{
			{Id: PostId("1"), Author: &user.User{Username: username}},
			{Id: PostId("2"), Author: &user.User{Username: username}},
		}

		mockMongoColl.EXPECT().
			Find(ctx, gomock.Any()).
			Return(mockFindResult, nil)
		mockFindResult.EXPECT().
			All(ctx, gomock.AssignableToTypeOf(&expectedPosts)).
			SetArg(1, expectedPosts).
			Return(nil)

		users, err := repo.GetUserPosts(context.Background(), username)
		assert.Nil(t, err)
		assert.Equal(t, []*Post{
			{Id: "1", Author: &user.User{Username: username}},
			{Id: "2", Author: &user.User{Username: username}},
		}, users)
	})
}
