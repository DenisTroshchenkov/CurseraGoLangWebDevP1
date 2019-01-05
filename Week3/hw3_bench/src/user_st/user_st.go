package user_st


//easyjson:json
type User struct {
	Browsers []string `json:"browsers"`
	Name string `json:"name"`
	Email string `json:"email"`
}
