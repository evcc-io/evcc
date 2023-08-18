package server

type product struct {
	Name     string `json:"name"`
	Template string `json:"template"`
	Group    string `json:"group,omitempty"`
}

type products []product
