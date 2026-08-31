package server

type product struct {
	Name     string `json:"name"`
	Template string `json:"template"`
	Group    string `json:"group,omitempty"`
	Link     string `json:"link,omitempty"`
}

type products []product
