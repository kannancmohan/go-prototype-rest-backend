package dto

type CreateUserReq struct {
	Username string
	Email    string
	Password string
	Role     string
}

type UpdateUserReq struct {
	ID       int64
	Username string
	Email    string
	Password string
	Role     string
}
