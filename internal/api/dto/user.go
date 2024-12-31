package dto

type CreateUserRequest struct {
	Username string
	Email    string
	Password string
	Role     string
}

type UpdateUserRequest struct {
	ID       int64
	Username string
	Email    string
	Password string
	Role     string
}
