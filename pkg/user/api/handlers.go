package api

import (
	"context"
	"fmt"
	"net/http"

	"crud/pkg/common"
	"crud/pkg/logger"
	"crud/pkg/user"
)

type (
	UserRepo interface {
		UserExists(string) bool
		GetByUsernameAndPass(string, string) (*user.User, error)
		Add(*user.User) (string, error)
	}

	SessionManager interface {
		CreateToken(*user.User) (string, error)
		CleanupUserSessions(userId string) error
	}

	UserHandler struct {
		Repo           UserRepo
		SessionManager SessionManager
	}

	HttpUser struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
)

func NewUserHanler(r UserRepo, sm SessionManager) *UserHandler {
	return &UserHandler{
		Repo:           r,
		SessionManager: sm,
	}
}

func (uh UserHandler) LogIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	httpUser := new(HttpUser)
	err := common.ParseReqBody(r.Body, httpUser)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as user: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	user, err := uh.Repo.GetByUsernameAndPass(httpUser.Username, httpUser.Password)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't get the user by username `%s` and password: %v",
			httpUser.Username, err)
		common.WriteMsg(w, "user not found", http.StatusNotFound)
		return
	}

	// Remove expired user session if there are any
	if err := uh.SessionManager.CleanupUserSessions(user.Id); err != nil {
		logger.Log(r.Context()).Errorf("user/handlers: can't cleanup sessions for user `%s`, %v", httpUser.Username, err)
		common.WriteMsg(w, "failed managing user sessions", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	uh.sendToken(w, user)
}

func (uh UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	httpUser := new(HttpUser)
	err := common.ParseReqBody(r.Body, httpUser)
	if err != nil {
		logger.Log(r.Context()).Errorf("can't parse request body as user: %v", err)
		common.WriteMsg(w, "bad request format", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	if uh.Repo.UserExists(httpUser.Username) {
		msg := fmt.Sprintf(`user "%s" already exists`, httpUser.Username)
		logger.Log(r.Context()).Error(msg)
		common.WriteMsg(w, msg, http.StatusConflict)
		return
	}

	salt := common.RandStringRunes(8)
	pass := common.HashPass(httpUser.Password, salt)
	user := &user.User{
		Username: httpUser.Username,
		Password: pass,
		// Id is handled below
	}
	id, err := uh.Repo.Add(user)
	if err != nil {
		common.WriteMsg(w, "can't add user", http.StatusInternalServerError)
		return
	}
	user.Id = id

	w.WriteHeader(http.StatusCreated)
	uh.sendToken(w, user)
}

func (uh *UserHandler) sendToken(w http.ResponseWriter, user *user.User) {
	token, err := uh.SessionManager.CreateToken(user)
	if err != nil {
		logger.Log(context.Background()).Errorf("can't create JWT token from user: %v", err)
		common.WriteMsg(w, "user authentication failed", http.StatusInternalServerError)
		return
	}

	tk := struct {
		Token string `json:"token"`
	}{token}
	common.WriteRespJSON(w, tk)
}
