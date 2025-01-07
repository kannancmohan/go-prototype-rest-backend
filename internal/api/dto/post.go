package dto

type CreatePostReq struct {
	Content string
	Title   string
	UserID  int64
	Tags    []string
}

type UpdatePostReq struct {
	ID      int64
	Content string
	Title   string
	Tags    []string
}
