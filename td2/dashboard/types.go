package dash

type ChainStatus struct {
	MsgType            string  `json:"msgType"`
	Name               string  `json:"name"`
	ChainId            string  `json:"chain_id"`
	Moniker            string  `json:"moniker"`
	Bonded             bool    `json:"bonded"`
	Jailed             bool    `json:"jailed"`
	Tombstoned         bool    `json:"tombstoned"`
	Missed             int64   `json:"missed"`
	Window             int64   `json:"window"`
	MinSignedPerWindow float64 `json:"min_signed_per_window"`
	Nodes              int     `json:"nodes"`
	HealthyNodes       int     `json:"healthy_nodes"`
	ActiveAlerts       int     `json:"active_alerts"`
	Height             int64   `json:"height"`
	LastError          string  `json:"last_error"`

	Blocks []int `json:"blocks"`
}

type LogMessage struct {
	MsgType string `json:"msgType"`
	Ts      int64  `json:"ts"`
	Msg     string `json:"msg"`
}
