package dto

type CreatePostRequest struct {
	Content string
	Title   string
	UserID  int64
	Tags    []string
}

type UpdatePostRequest struct {
	ID      int64
	Content string
	Title   string
	Tags    []string
}
