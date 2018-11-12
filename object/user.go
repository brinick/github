package object

// User is a Github account
type User struct {
	Login string `json:"login,omitempty"`
	ID    int    `json:"id,omitempty"`
}
