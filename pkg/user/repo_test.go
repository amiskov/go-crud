package user

import (
	"context"
	"fmt"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	. "crud/pkg/common"
)

var (
	userID     = "1"
	username   = "pike"
	password   = "sdfsdfsdf"
	salt       = "12345678"
	hashedPass = HashPass(password, salt)
)

func TestGetById(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	r := NewUserRepo(db)

	t.Run("should return user", func(t *testing.T) {
		expect := &User{Id: userID, Username: username}

		rows := sqlmock.NewRows([]string{"id", "username"})
		rows.AddRow(expect.Id, expect.Username)

		mock.
			ExpectQuery("SELECT id, username FROM users where").
			WithArgs(userID).
			WillReturnRows(rows)

		gotUser, err := r.GetById(context.TODO(), userID)
		if err != nil {
			t.Errorf("unexpected err: %s", err)
			return
		}
		assert.Equal(t, expect, gotUser)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return DB error", func(t *testing.T) {
		expectedErr := fmt.Errorf("mock_db_error")
		mock.
			ExpectQuery("SELECT id, username FROM users where").
			WithArgs(userID).
			WillReturnError(expectedErr)
		_, err = r.GetById(context.TODO(), userID)
		assert.ErrorIs(t, err, expectedErr)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})
}

func TestRepoAdd(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()
	repo := NewUserRepo(db)
	testUser := &User{Id: userID, Username: username, Password: hashedPass}

	t.Run("should add new user", func(t *testing.T) {
		mock.
			ExpectExec("INSERT INTO users").
			WithArgs(username, hashedPass).
			WillReturnResult(sqlmock.NewResult(1, 1))

		addedUserId, err := repo.Add(testUser)
		if err != nil {
			t.Errorf("unexpected error %s", err)
			return
		}
		assert.Equal(t, addedUserId, userID)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return query error", func(t *testing.T) {
		expectedErr := fmt.Errorf("bad query")
		mock.
			ExpectExec("INSERT INTO users").
			WithArgs(username, hashedPass).
			WillReturnError(expectedErr)
		_, err = repo.Add(testUser)
		assert.ErrorIs(t, err, expectedErr)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return result error", func(t *testing.T) {
		expectedErr := fmt.Errorf("bad_result")
		mock.
			ExpectExec("INSERT INTO users").
			WithArgs(username, hashedPass).
			WillReturnResult(sqlmock.NewErrorResult(expectedErr))

		_, err := repo.Add(testUser)
		assert.ErrorIs(t, err, expectedErr)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return zero LastInsertId/RowsAffected error", func(t *testing.T) {
		mock.
			ExpectExec("INSERT INTO users").
			WithArgs(username, hashedPass).
			WillReturnResult(sqlmock.NewResult(0, 0))
		_, err = repo.Add(testUser)
		assert.ErrorContains(t, err, "user wasn't added")
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})
}

func TestGetByUsernameAndPass(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()
	r := NewUserRepo(db)
	expect := &User{Id: userID, Username: username, Password: hashedPass}

	t.Run("should return user", func(t *testing.T) {
		row := sqlmock.NewRows([]string{"id", "username", "password"}).
			AddRow(expect.Id, expect.Username, expect.Password)
		mock.
			ExpectQuery("SELECT id, username, password FROM users where username").
			WithArgs(username).
			WillReturnRows(row)

		gotUser, err := r.GetByUsernameAndPass(username, password)
		if err != nil {
			t.Errorf("unexpected err: %s", err)
			return
		}
		assert.Equal(t, expect, gotUser)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return error: bad password", func(t *testing.T) {
		row := sqlmock.NewRows([]string{"id", "username", "password"}).
			AddRow(expect.Id, expect.Username, expect.Password)
		mock.
			ExpectQuery("SELECT id, username, password FROM users where username").
			WithArgs(username).
			WillReturnRows(row)
		_, err := r.GetByUsernameAndPass(username, "badpassword")
		assert.ErrorContains(t, err, "password is invalid")
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return error: DB error", func(t *testing.T) {
		expectedErr := fmt.Errorf("mock_db_error")
		mock.
			ExpectQuery("SELECT id, username, password FROM users where username").
			WithArgs(username).
			WillReturnError(expectedErr)
		_, err = r.GetByUsernameAndPass(username, password)
		assert.ErrorIs(t, err, expectedErr)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})
}

func TestUserExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()
	r := NewUserRepo(db)

	t.Run("should return true", func(t *testing.T) {
		existingUser := &User{Id: userID}
		rows := sqlmock.NewRows([]string{"id"})
		rows.AddRow(existingUser.Id)
		mock.
			ExpectQuery("SELECT id FROM users where").
			WithArgs(username).
			WillReturnRows(rows)
		exists := r.UserExists(username)
		assert.Equal(t, exists, true)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return false", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "username"}).AddRow("2", "test")
		mock.
			ExpectQuery("SELECT id FROM users where").
			WithArgs(username).
			WillReturnRows(rows)
		exists := r.UserExists(username)
		assert.Equal(t, exists, false)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})
}

func TestGetAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()
	r := NewUserRepo(db)

	t.Run("should return users", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "username", "password"})
		expectedUsers := []*User{
			{Id: "1", Username: "user1", Password: hashedPass},
			{Id: "2", Username: "user2", Password: hashedPass},
			{Id: "3", Username: "user3", Password: hashedPass},
		}
		for _, u := range expectedUsers {
			rows.AddRow(u.Id, u.Username, u.Password)
		}
		mock.
			ExpectQuery("SELECT id, username, password FROM users").
			WillReturnRows(rows)
		gotUsers, err := r.GetAll()
		if err != nil {
			t.Errorf("unexpected err: %s", err)
			return
		}
		assert.Equal(t, expectedUsers, gotUsers)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return DB error", func(t *testing.T) {
		expectedErr := fmt.Errorf("mock_db_error")
		mock.
			ExpectQuery("SELECT id, username, password FROM users").
			WillReturnError(expectedErr)
		_, err = r.GetAll()
		assert.ErrorIs(t, err, expectedErr)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})

	t.Run("should return scan rows error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow("2")
		mock.
			ExpectQuery("SELECT id, username, password FROM users").
			WillReturnRows(rows)
		_, err = r.GetAll()
		assert.ErrorContains(t, err, "scan")
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations unfulfilled: %s", err)
			return
		}
	})
}
