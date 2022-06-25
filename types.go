package tenderduty

import (
	"context"
	"fmt"
	dash "github.com/blockpane/tenderduty/dashboard"
	"github.com/go-yaml/yaml"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"
)

const showBLocks = 512

// Config holds both the settings for tenderduty to monitor and state information while running.
type Config struct {
	alertChan  chan *alertMsg // channel used for outgoing notifications
	updateChan chan *dash.ChainStatus
	logChan    chan dash.LogMessage
	statsChan  chan *promUpdate
	ctx        context.Context
	cancel     context.CancelFunc

	// EnableDash enables the web dashboard
	EnableDash bool `yaml:"enable_dashboard"`
	// Listen is the URL for the dashboard to listen on, must be a valid/parsable URL
	Listen      string `yaml:"listen_port"`
	HideLogs    bool   `yaml:"hide_logs"`
	NodeDownMin int    `yaml:"node_down_alert_minutes"`

	Prom                 bool `yaml:"prometheus_enabled"`
	PrometheusListenPort int  `yaml:"prometheus_listen_port"`

	// Pagerduty configuration values
	Pagerduty PDConfig `yaml:"pagerduty"`
	// Discord webhook information
	Discord DiscordConfig `yaml:"discord"`
	// Telegram webhook information
	Telegram TeleConfig `yaml:"telegram"`

	chainsMux sync.RWMutex // prevents concurrent map access for Chains
	// Chains has settings for each validator to monitor. The map's name does not need to match the chain-id.
	Chains map[string]*ChainConfig `yaml:"chains"`
}

// ChainConfig represents a validator to be monitored on a chain, it is somewhat of a misnomer since multiple
// validators can be monitored on a single chain.
type ChainConfig struct {
	name           string
	wsclient       *TmConn       // custom websocket client to work around wss:// bugs in tendermint
	client         *rpchttp.HTTP // legit tendermint client
	noNodes        bool          // tracks if all nodes are down
	noBlocksCount  int           // counter (minutes) of how long since a block has been seen
	valInfo        *ValInfo      // recent validator state, only refreshed every few minutes
	lastValInfo    *ValInfo      // use for detecting newly-jailed/tombstone
	blocksResults  []int
	lastError      string
	lastBlockTime  time.Time
	lastBlockAlarm bool
	lastBlockNum   int64
	activeAlerts   int

	statTotalSigns      float64
	statTotalProps      float64
	statTotalMiss       float64
	statPrevoteMiss     float64
	statPrecommitMiss   float64
	statConsecutiveMiss float64

	// ChainId is used to ensure any endpoints contacted claim to be on the correct chain. This is a weak verification,
	// no light client validation is performed, so caution is advised when using public endpoints.
	ChainId string `yaml:"chain_id"`
	// ValAddress is the validator operator address to be monitored. Tenderduty v1 required the consensus address,
	// this is no longer needed. The operator address is much easier to find in explorers etc.
	ValAddress string `yaml:"valoper_address"`
	// ExtraInfo will be appended to the alert data. This is useful for pagerduty because multiple tenderduty instances
	// can be pointed at pagerduty and duplicate alerts will be filtered by using a key. The first alert will win, this
	// can be useful for knowing what tenderduty instance sent the alert.
	ExtraInfo string `yaml:"extra_info"`
	// Alerts defines the types of alerts to send for this chain.
	Alerts AlertConfig `yaml:"alerts"`
	// PublicFallback determines if tenderduty should attempt to use public RPC endpoints in the situation that not
	// explicitly defined RPC servers are available. Not recommended.
	PublicFallback bool `yaml:"public_fallback"`
	// Nodes defines what RPC servers to connect to.
	Nodes []*NodeConfig `yaml:"nodes"`
}

func (cc *ChainConfig) mkUpdate(t metricType, v float64, node string) *promUpdate {
	return &promUpdate{
		metric:   t,
		counter:  v,
		name:     cc.name,
		chainId:  cc.ChainId,
		moniker:  cc.valInfo.Moniker,
		endpoint: node,
	}
}

// AlertConfig defines the type of alerts to send for a ChainConfig
type AlertConfig struct {
	// How many minutes to wait before alerting that no new blocks have been seen
	Stalled int `yaml:"stalled_minutes"`
	// Whether to alert when no new blocks are seen
	StalledAlerts bool `yaml:"stalled_enabled"`

	// How many missed blocks are acceptable before alerting
	ConsecutiveMissed int `yaml:"consecutive_missed"`
	// Tag for pagerduty to set the alert priority
	ConsecutivePriority string `yaml:"consecutive_priority"`
	// Whether to alert on consecutive missed blocks
	ConsecutiveAlerts bool `yaml:"consecutive_enabled"`

	// Window is how many blocks missed as a percentage of the slashing window to trigger an alert
	Window int `yaml:"percentage_missed"`
	// PercentagePriority is a tag for pagerduty to route on priority
	PercentagePriority string `yaml:"percentage_priority"`
	// PercentageAlerts is whether to alert on percentage based misses
	PercentageAlerts bool `yaml:"percentage_enabled"`

	// AlertIfInactive decides if tenderduty send an alert if the validator is not in the active set?
	AlertIfInactive bool `yaml:"alert_if_inactive"`
	// AlertIfNoServers: should an alert be sent if no servers are reachable?
	AlertIfNoServers bool `yaml:"alert_if_no_servers"`
	// PagerdutyAlerts: Should pagerduty alerts be sent for this chain? Both 'config.pagerduty.enabled: yes' and this must be set.
	PagerdutyAlerts bool `yaml:"pagerduty_alerts"`
	// DiscordAlerts: Should discord alerts be sent for this chain? Both 'config.discord.enabled: yes' and this must be set.
	DiscordAlerts bool `yaml:"discord_alerts"`
	// TelegramAlerts: Should telegram alerts be sent for this chain? Both 'config.telegram.enabled: yes' and this must be set.
	TelegramAlerts bool `yaml:"telegram_alerts"`
}

// NodeConfig holds the basic information for a node to connect to.
type NodeConfig struct {
	Url         string `yaml:"url"`
	AlertIfDown bool   `yaml:"alert_if_down"`

	down      bool
	syncing   bool
	lastMsg   string
	downSince time.Time
}

// PDConfig is the information required to send alerts to PagerDuty
type PDConfig struct {
	Enabled         bool   `yaml:"enabled"`
	ApiKey          string `yaml:"api_key"`
	DefaultSeverity string `yaml:"default_severity"`
}

// DiscordConfig holds the information needed to publish to a Discord webhook for sending alerts
type DiscordConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Webhook  string   `yaml:"webhook"`
	Mentions []string `yaml:"mentions"`
	// TODO
}

// TeleConfig holds the information needed to publish to a Telegram webhook for sending alerts
type TeleConfig struct {
	Enabled  bool     `yaml:"enabled"`
	ApiKey   string   `yaml:"api_key"`
	Channel  string   `yaml:"channel"`
	Mentions []string `yaml:"mentions"`
	// TODO
}

// validateConfig is a non-exhaustive check for common problems with the configuration. Needs love.
func validateConfig(c *Config) (fatal bool, problems []string) {
	problems = make([]string, 0)
	var err error

	if c.EnableDash {
		_, err = url.Parse(c.Listen)
		if err != nil {
			fatal = true
			problems = append(problems, fmt.Sprintf("error: The listen URL %s does not appear to be valid", c.Listen))
		}
	}

	if c.Pagerduty.Enabled {
		rex := regexp.MustCompile(`[+_-]`)
		if rex.MatchString(c.Pagerduty.ApiKey) {
			fatal = true
			problems = append(problems, "error: The Pagerduty key provided appears to be an Oauth token, not a V2 Events API key.")
		}
	}

	if c.Discord.Enabled {
		// TODO
	}

	if c.Telegram.Enabled {
		// TODO
	}

	for k, v := range c.Chains {
		if v.blocksResults == nil {
			v.blocksResults = make([]int, showBLocks)
			for i := range v.blocksResults {
				v.blocksResults[i] = -1
			}
		}
		if v.name == "" {
			v.name = k
		}

		v.valInfo = &ValInfo{Moniker: "not connected"}
		switch true {
		case v.Alerts.DiscordAlerts && !c.Discord.Enabled:
			problems = append(problems, fmt.Sprintf("warn: %s is configured for discord alerts, but it is not enabled", k))
			fallthrough
		case v.Alerts.PagerdutyAlerts && !c.Pagerduty.Enabled:
			problems = append(problems, fmt.Sprintf("warn: %s is configured for pagerduty alerts, but it is not enabled", k))
			fallthrough
		case v.Alerts.TelegramAlerts && !c.Telegram.Enabled:
			problems = append(problems, fmt.Sprintf("warn: %s is configured for telegram alerts, but it is not enabled", k))
		case !v.Alerts.ConsecutiveAlerts && !v.Alerts.PercentageAlerts && !v.Alerts.AlertIfInactive && !v.Alerts.AlertIfNoServers:
			problems = append(problems, fmt.Sprintf("warn: %s has no alert types configured", k))
			fallthrough
		case !v.Alerts.PagerdutyAlerts && !v.Alerts.DiscordAlerts && !v.Alerts.TelegramAlerts:
			problems = append(problems, fmt.Sprintf("warn: %s has no notifications configured", k))
		}
		td.updateChan <- &dash.ChainStatus{
			MsgType:      "status",
			Name:         v.name,
			ChainId:      v.ChainId,
			Moniker:      v.valInfo.Moniker,
			Bonded:       v.valInfo.Bonded,
			Jailed:       v.valInfo.Jailed,
			Tombstoned:   v.valInfo.Tombstoned,
			Missed:       v.valInfo.Missed,
			Window:       v.valInfo.Window,
			Nodes:        len(v.Nodes),
			HealthyNodes: 0,
			ActiveAlerts: 0,
			Blocks:       v.blocksResults,
		}

	}
	return
}

// loadConfig creates a new Config from a file.
func loadConfig(yamlFile string) (*Config, error) {
	f, e := os.OpenFile(yamlFile, os.O_RDONLY, 0644)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	i, e := f.Stat()
	if e != nil {
		return nil, e
	}
	b := make([]byte, int(i.Size()))
	_, e = f.Read(b)
	if e != nil {
		return nil, e
	}
	c := &Config{}
	e = yaml.Unmarshal(b, c)
	if e != nil {
		return nil, e
	}

	c.alertChan = make(chan *alertMsg)
	c.logChan = make(chan dash.LogMessage, 32)
	c.updateChan = make(chan *dash.ChainStatus, 32)
	c.statsChan = make(chan *promUpdate, 32)
	c.ctx, c.cancel = context.WithCancel(context.Background())

	return c, nil
}
