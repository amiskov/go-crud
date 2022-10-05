package user

import (
	"bytes"
	"context"
	"crud/pkg/common"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func (r *UserRepo) Add(u *User) (string, error) {
	result, err := r.db.Exec("INSERT INTO users(username, password) VALUES($1, $2)", u.Username, u.Password)
	if err != nil {
		return ``, err
	}
	userID, lastIdErr := result.LastInsertId()
	if lastIdErr != nil {
		return ``, fmt.Errorf("user/repo: user wasn't added: %w", lastIdErr)
	}
	if userID == 0 {
		return ``, fmt.Errorf("user/repo: user wasn't added, LastInsertId is 0")
	}
	return strconv.FormatInt(userID, 10), nil
}

func (r *UserRepo) GetByUsernameAndPass(uname string, pass string) (*User, error) {
	row := r.db.QueryRow("SELECT id, username, password FROM users where username=$1", uname)
	u := new(User)
	if err := row.Scan(&u.Id, &u.Username, &u.Password); err != nil {
		return nil, fmt.Errorf("user/repo: row scan failed: %w", err)
	}
	// User found by username, now check if passwords are the same
	salt := string(u.Password[0:8])
	if !bytes.Equal(common.HashPass(pass, salt), u.Password) {
		return nil, errors.New("user/repo: password is invalid")
	}
	return u, nil
}

func (r *UserRepo) UserExists(uname string) bool {
	row := r.db.QueryRow("SELECT id FROM users where username=$1", uname)
	u := new(User)
	if err := row.Scan(&u.Id); err != nil {
		log.Printf("user/repo: could not scan row: %v", err)
		return false
	}
	return true
}

func (r *UserRepo) GetById(ctx context.Context, uid string) (*User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, username FROM users where id=$1", uid)
	u := new(User)
	if err := row.Scan(&u.Id, &u.Username); err != nil {
		return u, fmt.Errorf("user/repo: could not scan row: %w", err)
	}
	return u, nil
}

// Returns all users. Used only for seeding the DB.
func (r *UserRepo) GetAll() ([]*User, error) {
	rows, err := r.db.Query("SELECT id, username, password FROM users")
	if err != nil {
		return nil, fmt.Errorf("repo: failed executing query for getting all users: %w", err)
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		u := new(User)
		err := rows.Scan(&u.Id, &u.Username, &u.Password)
		if err != nil {
			return nil, fmt.Errorf("user/repo: could not scan row: %w", err)
		}
		users = append(users, u)
	}

	return users, nil
}
