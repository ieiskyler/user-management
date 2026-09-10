package handler

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

// listUsersQuery holds pagination parameters for listing users.
type ListUsersQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

// Normalize fills in sane defaults and clamps values into acceptable ranges.
func (q *ListUsersQuery) Normalize() {
	if q.Page < 1 {
		q.Page = defaultPage
	}

	if q.Limit < 1 {
		q.Limit = defaultLimit
	}
}
