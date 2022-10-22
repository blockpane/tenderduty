package tenderduty

import (
	"context"
	"errors"
	"fmt"
	oracle "github.com/Team-Kujira/core/x/oracle/types"
	dash "github.com/blockpane/tenderduty/v2/td2/dashboard"
	banking "github.com/cosmos/cosmos-sdk/x/bank/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// newRpc sets up the rpc client used for monitoring. It will try nodes in order until a working node is found.
// it will also get some initial info on the validator's status.
func (cc *ChainConfig) newRpc() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
		status, err := cc.client.Status(ctx)
		if err != nil {
			msg = fmt.Sprintf("❌ could not get status for %s: (%s) %s", cc.name, u, err)
			down = true
			l(msg)
			return
		}
		if status.NodeInfo.Network != cc.ChainId {
			msg = fmt.Sprintf("chain id %s on %s does not match, expected %s, skipping", status.NodeInfo.Network, u, cc.ChainId)
			down = true
			l(msg)
			return
		}
		if status.SyncInfo.CatchingUp {
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
	cc.lastError = "no usable RPC endpoints available for " + cc.ChainId
	if td.EnableDash {
		td.updateChan <- &dash.ChainStatus{
			MsgType:      "status",
			Name:         cc.name,
			ChainId:      cc.ChainId,
			Moniker:      cc.valInfo.Moniker,
			Bonded:       cc.valInfo.Bonded,
			Jailed:       cc.valInfo.Jailed,
			Tombstoned:   cc.valInfo.Tombstoned,
			Missed:       cc.valInfo.Missed,
			Window:       cc.valInfo.Window,
			Nodes:        len(cc.Nodes),
			HealthyNodes: 0,
			ActiveAlerts: 1,
			Height:       0,
			LastError:    cc.lastError,
			Blocks:       cc.blocksResults,
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
					cwt, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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
					Moniker:      cc.valInfo.Moniker,
					Bonded:       cc.valInfo.Bonded,
					Jailed:       cc.valInfo.Jailed,
					Tombstoned:   cc.valInfo.Tombstoned,
					Missed:       cc.valInfo.Missed,
					Window:       cc.valInfo.Window,
					Conspub:      cc.valInfo.Conspub,
					Valcons:      cc.valInfo.Valcons,
					OracleMissed: cc.valInfo.OracleMissed,
				}
			}
			err = cc.GetValInfo(false)
			if err != nil {
				l("❓ refreshing signing info for", cc.ValAddress, err)
			}
			for _, wallet := range cc.Wallets {
				go func(wallet *WalletConfig) {
					qParams := banking.QueryBalanceRequest{Address: wallet.WalletAddress, Denom: wallet.WalletDenom}
					b, err := qParams.Marshal()
					resp, err := cc.client.ABCIQuery(ctx, "/cosmos.bank.v1beta1.Query/Balance", b)
					if resp == nil || resp.Response.Value == nil {
						err = errors.New(fmt.Sprintf("🛑 could not get wallet balance for %s, got empty response", wallet.WalletName))
						return
					}
					params := &banking.QueryBalanceResponse{}
					err = params.Unmarshal(resp.Response.Value)
					if err != nil {
						return
					}
					wallet.recorded = true
					wallet.balance = params.Balance.Amount.Int64()

					if wallet.balance < wallet.WalletMinimumBalance {
						l(fmt.Sprintf("❌ %s/%s %d > %d wallet balance below threshold", wallet.WalletName, wallet.WalletAddress, wallet.WalletMinimumBalance, wallet.balance))
					} else {
						l(fmt.Sprintf("OK %s/%s %d <= %d wallet balance above threshold", wallet.WalletName, wallet.WalletAddress, wallet.WalletMinimumBalance, wallet.balance))
					}

				}(wallet)
			}
			if cc.KujiraPriceOracle {
				qParams := oracle.QueryMissCounterRequest{ValidatorAddr: cc.ValAddress}
				b, err := qParams.Marshal()
				resp, err := cc.client.ABCIQuery(ctx, "/kujira.oracle.Query/MissCounter", b)
				if err != nil {
					l(err.Error())
					return
				}
				if resp == nil || resp.Response.Value == nil {
					err = errors.New("🛑 could not get oracle Miss counter, got empty response")
					return
				}

				params := &oracle.QueryMissCounterResponse{}
				err = params.Unmarshal(resp.Response.Value)
				if err != nil {
					return
				}
				if params.MissCounter > cc.lastValInfo.OracleMissed {
					l(fmt.Sprintf("❌ miss count %d %d", params.MissCounter, cc.valInfo.OracleMissed))
				} else {
					l(fmt.Sprintf("OK miss count %d %d", params.MissCounter, cc.valInfo.OracleMissed))
				}
				cc.valInfo.OracleMissed = params.MissCounter
			} else {
				cc.lastOracleMissAlert = 0
				cc.valInfo.OracleMissed = 0
			}
		}
	}
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
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return u
	}
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
