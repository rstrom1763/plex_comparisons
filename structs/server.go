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
