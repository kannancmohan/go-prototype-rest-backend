package model

type Post struct {
	ID        int64    `json:"id"`
	Content   string   `json:"content"`
	Title     string   `json:"title"`
	UserID    int64    `json:"user_id"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	Version   int      `json:"version"`
	//Comments  []Comment `json:"comments"`
	User User `json:"user"`
}

func (p *Post) BasicSanityCheck() bool {
	return p != nil && p.ID > 0
}
