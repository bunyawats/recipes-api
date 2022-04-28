package models

// swagger:parameters auth signIn
type User struct {
	// User's password
	//
	// required: true
	Password string `json:"password"`
	// User's login
	//
	// required: true
	Username string `json:"username"`
}
