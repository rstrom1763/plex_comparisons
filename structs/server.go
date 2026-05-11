package structs

type Server struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Health  string `json:"health,omitempty"`
	Token   string `json:"token,omitempty"`
}

type AllowListedServer struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

type SavedFilter struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	FilterData string `json:"filter_data"` // Base64 encoded JSON
}

type TrustedServer struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	TokenHash string `json:"-"`
}

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}
