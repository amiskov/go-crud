package user

type User struct {
	Username string `json:"username"`
	Password []byte `json:"-"`
	Id       string `json:"id"`
}

type UserFromToken struct {
	Username string `json:"username"`
	Id       string `json:"id"`
}
