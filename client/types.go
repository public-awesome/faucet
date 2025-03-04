package client

type AccountInfoResponse struct {
	AccountInfo AccountInfo `json:"info"`
}
type PubKey struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}
type AccountInfo struct {
	Address       string `json:"address"`
	PubKey        PubKey `json:"pub_key"`
	AccountNumber string `json:"account_number"`
	Sequence      string `json:"sequence"`
}
