package api

import (
	"bytes"
	"crud/pkg/common"
	"crud/pkg/logger"
	"crud/pkg/middleware"
	"crud/pkg/user"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gomock "github.com/golang/mock/gomock"
)

var (
	userId         = "1"
	username       = "pike"
	salt           = "12345678"
	password       = "sdfsdfsdf"
	hashedPassword = common.HashPass("sdfsdfsdf", salt)
	// JWT for the user details above
	jwtToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjp7InVzZXJuYW1lIjoicGlrZSIsImlkIjoiMSJ9LCJleHAiOjE2Njk4ODgzMjksImp0aSI6InZZeVFQYUhRRlEiLCJpYXQiOjE2NjIxMTIzMjl9.FJq9VmKF_j4JCRG4Pf4gjhKxaGPyv916tnPwKmunn44"
)

func TestLogIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	existingUser := user.User{Id: userId, Username: username, Password: hashedPassword}
	mockRepo := NewMockUserRepo(ctrl)
	mockSm := NewMockSessionManager(ctrl)
	mockService := &UserHandler{
		Repo:           mockRepo,
		SessionManager: mockSm,
	}

	// Add AccessLog middleware for `/login` because we use it in handler methods
	logMiddleware := middleware.NewLoggingMiddleware(logger.Run("fatal"))
	testServer := httptest.NewServer(logMiddleware.AccessLog(http.HandlerFunc(mockService.LogIn)))

	loginReq := func(un, pw, url string) *http.Request {
		body := strings.NewReader(`{"username": "` + un + `", "password": "` + pw + `"}`)
		return httptest.NewRequest("POST", url, body)
	}

	t.Run("login is OK", func(t *testing.T) {
		mockRepo.EXPECT().GetByUsernameAndPass(username, password).Return(&existingUser, nil)
		mockSm.EXPECT().CreateToken(&existingUser).Return(jwtToken, nil)

		w := httptest.NewRecorder()
		mockService.LogIn(w, loginReq(username, password, testServer.URL))
		resp := w.Result()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("error reading login response body")
			return
		}

		if !bytes.Contains(body, []byte(jwtToken)) {
			t.Errorf("login response doesn't contain JWT token")
			return
		}
	})

	t.Run("user not found", func(t *testing.T) {
		badUsername, badPassword := "notexists", "nevermind"
		mockRepo.EXPECT().GetByUsernameAndPass(badUsername, badPassword).
			Return(nil, fmt.Errorf("user not found"))
		w := httptest.NewRecorder()
		mockService.LogIn(w, loginReq(badUsername, badPassword, testServer.URL))
		badResp := w.Result()
		if badResp.StatusCode != 404 {
			t.Errorf("expected 404, got %d", badResp.StatusCode)
			return
		}
	})
}
