package iobroker

type StateResponse struct {
	ID     string `json:"id"`
	VAL    any    `json:"val"`
	Q      int    `json:"q"`
	Ts     int    `json:"ts"`
	Lc     int    `json:"lc"`
	Ack    bool   `json:"ack"`
	From   string `json:"from"`
	Expire int    `json:"expire"`
	Type   string `json:"type"`
}

type SetStateResponse struct {
	Error  string `json:"error"`
	Result string `json:"result"`
}

type SetValueRequest struct {
	Value any  `json':"value"`
	Ack   bool `json':"ack"`
}
