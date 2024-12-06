package tenderduty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	dash "github.com/blockpane/tenderduty/v2/td2/dashboard"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

// newRpc sets up the rpc client used for monitoring. It will try nodes in order until a working node is found.
// it will also get some initial info on the validator's status.
func (cc *ChainConfig) newRpc() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var anyWorking bool // if healthchecks are running, we will skip to the first known good node.
	for _, endpoint := range cc.Nodes {
		anyWorking = anyWorking || !endpoint.down
	}

	// grab the first working endpoint
	tryUrl := func(u string) (msg string, down, syncing bool) {
		_, err := url.Parse(u)
		if err != nil {
			msg = fmt.Sprintf("❌ could not parse url %s: (%s) %s", cc.name, u, err)
			l(msg)
			down = true
			return
		}
		cc.client, err = rpchttp.New(u, "/websocket")
		if err != nil {
			msg = fmt.Sprintf("❌ could not connect client for %s: (%s) %s", cc.name, u, err)
			l(msg)
			down = true
			return
		}
		var network string
		var catching_up bool
		status, err := cc.client.Status(ctx)
		if err != nil {
			n, c, err := getStatusWithEndpoint(ctx, u)
			if err != nil {
				msg = fmt.Sprintf("❌ could not get status for %s: (%s) %s", cc.name, u, err)
				down = true
				l(msg)
				return
			}
			network, catching_up = n, c
		} else {
			network, catching_up = status.NodeInfo.Network, status.SyncInfo.CatchingUp
		}
		if network != cc.ChainId {
			msg = fmt.Sprintf("chain id %s on %s does not match, expected %s, skipping", network, u, cc.ChainId)
			down = true
			l(msg)
			return
		}
		if catching_up {
			msg = fmt.Sprint("🐢 node is not synced, skipping ", u)
			syncing = true
			down = true
			l(msg)
			return
		}
		cc.noNodes = false
		return
	}
	down := func(endpoint *NodeConfig, msg string) {
		if !endpoint.down {
			endpoint.down = true
			endpoint.downSince = time.Now()
		}
		endpoint.lastMsg = msg
	}
	for _, endpoint := range cc.Nodes {
		if anyWorking && endpoint.down {
			continue
		}
		if msg, failed, syncing := tryUrl(endpoint.Url); failed {
			endpoint.syncing = syncing
			down(endpoint, msg)
			continue
		}
		return nil
	}
	if cc.PublicFallback {
		if u, ok := getRegistryUrl(cc.ChainId); ok {
			node := guessPublicEndpoint(u)
			l(cc.ChainId, "⛑ attemtping to use public fallback node", node)
			if _, kk, _ := tryUrl(node); !kk {
				l(cc.ChainId, "⛑ connected to public endpoint", node)
				return nil
			}
		} else {
			l("could not find a public endpoint for", cc.ChainId)
		}
	}
	cc.noNodes = true
	alarms.clearAll(cc.name)
	cc.lastError = "no usable RPC endpoints available for " + cc.ChainId
	if td.EnableDash {
		td.updateChan <- &dash.ChainStatus{
			MsgType:            "status",
			Name:               cc.name,
			ChainId:            cc.ChainId,
			Moniker:            cc.valInfo.Moniker,
			Bonded:             cc.valInfo.Bonded,
			Jailed:             cc.valInfo.Jailed,
			Tombstoned:         cc.valInfo.Tombstoned,
			Missed:             cc.valInfo.Missed,
			Window:             cc.valInfo.Window,
			MinSignedPerWindow: cc.minSignedPerWindow,
			Nodes:              len(cc.Nodes),
			HealthyNodes:       0,
			ActiveAlerts:       1,
			Height:             0,
			LastError:          cc.lastError,
			Blocks:             cc.blocksResults,
		}
	}
	return errors.New("no usable endpoints available for " + cc.ChainId)
}

func (cc *ChainConfig) monitorHealth(ctx context.Context, chainName string) {
	tick := time.NewTicker(time.Minute)
	if cc.client == nil {
		_ = cc.newRpc()
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-tick.C:
			var err error
			for _, node := range cc.Nodes {
				go func(node *NodeConfig) {
					alert := func(msg string) {
						node.lastMsg = fmt.Sprintf("%-12s node %s is %s", chainName, node.Url, msg)
						if !node.AlertIfDown {
							// even if we aren't alerting, we want to display the status in the dashboard.
							node.down = true
							return
						}
						if !node.down {
							node.down = true
							node.downSince = time.Now()
						}
						if td.Prom {
							td.statsChan <- cc.mkUpdate(metricNodeDownSeconds, time.Since(node.downSince).Seconds(), node.Url)
						}
						l("⚠️ " + node.lastMsg)
					}
					c, e := rpchttp.New(node.Url, "/websocket")
					if e != nil {
						alert(e.Error())
					}
					cwt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					status, e := c.Status(cwt)
					cancel()
					if e != nil {
						alert("down")
						return
					}
					if status.NodeInfo.Network != cc.ChainId {
						alert("on the wrong network")
						return
					}
					if status.SyncInfo.CatchingUp {
						alert("not synced")
						node.syncing = true
						return
					}

					// node's OK, clear the note
					if node.down {
						node.lastMsg = ""
						node.wasDown = true
					}
					td.statsChan <- cc.mkUpdate(metricNodeDownSeconds, 0, node.Url)
					node.down = false
					node.syncing = false
					node.downSince = time.Unix(0, 0)
					cc.noNodes = false
					l(fmt.Sprintf("🟢 %-12s node %s is healthy", chainName, node.Url))
				}(node)
			}

			if cc.client == nil {
				e := cc.newRpc()
				if e != nil {
					l("💥", cc.ChainId, e)
				}
			}
			if cc.valInfo != nil {
				cc.lastValInfo = &ValInfo{
					Moniker:    cc.valInfo.Moniker,
					Bonded:     cc.valInfo.Bonded,
					Jailed:     cc.valInfo.Jailed,
					Tombstoned: cc.valInfo.Tombstoned,
					Missed:     cc.valInfo.Missed,
					Window:     cc.valInfo.Window,
					Conspub:    cc.valInfo.Conspub,
					Valcons:    cc.valInfo.Valcons,
				}
			}
			err = cc.GetValInfo(false)
			if err != nil {
				l("❓ refreshing signing info for", cc.ValAddress, err)
			}
		}
	}
}

func (c *Config) pingHealthcheck() {
	if !c.Healthcheck.Enabled {
		return
	}

	ticker := time.NewTicker(c.Healthcheck.PingRate * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				_, err := http.Get(c.Healthcheck.PingURL)
				if err != nil {
					l(fmt.Sprintf("❌ Failed to ping healthcheck URL: %s", err.Error()))
				} else {
					l(fmt.Sprintf("🏓 Successfully pinged healthcheck URL: %s", c.Healthcheck.PingURL))
				}
			}
		}
	}()
}

// endpointRex matches the first a tag's hostname and port if present.
var endpointRex = regexp.MustCompile(`//([^/:]+)(:\d+)?`)

// guessPublicEndpoint attempts to deal with a shortcoming in the tendermint RPC client that doesn't allow path prefixes.
// The cosmos.directory requires them. This is a workaround to get the actual URL for the server behind their proxy.
// The RPC base URL will return links endpoints, and we can parse this to guess the original URL.
func guessPublicEndpoint(u string) string {
	resp, err := http.Get(u + "/")
	if err != nil {
		return u
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return u
	}
	_ = resp.Body.Close()
	matches := endpointRex.FindStringSubmatch(string(b))
	if len(matches) < 2 {
		// didn't work
		return u
	}
	proto := "https://"
	port := ":443"
	// will be 3 elements if there is a port no port means listening on https
	if len(matches) == 3 && matches[2] != "" && matches[2] != ":443" {
		proto = "http://"
		port = matches[2]
	}
	return proto + matches[1] + port
}

func getStatusWithEndpoint(ctx context.Context, u string) (string, bool, error) {
	// Parse the URL
	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", false, err
	}

	// Check if the scheme is 'tcp' and modify to 'http'
	if parsedURL.Scheme == "tcp" {
		parsedURL.Scheme = "http"
	}

	queryPath := fmt.Sprintf("%s/status", parsedURL.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryPath, nil)
	if err != nil {
		return "", false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	type tendermintStatus struct {
		JsonRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			NodeInfo struct {
				Network string `json:"network"`
			} `json:"node_info"`
			SyncInfo struct {
				CatchingUp bool `json:"catching_up"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	var status tendermintStatus
	if err := json.Unmarshal(b, &status); err != nil {
		return "", false, err
	}
	return status.Result.NodeInfo.Network, status.Result.SyncInfo.CatchingUp, nil
}
