package tenderduty

import (
	"context"
	"errors"
	"fmt"
	dash "github.com/blockpane/tenderduty/dashboard"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"time"
)

func (cc *ChainConfig) newRpc() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// grab the first working endpoint
	for _, endpoint := range cc.Nodes {
		down := func(msg string) {
			if !endpoint.down {
				endpoint.down = true
				endpoint.downSince = time.Now()
			}
			endpoint.lastMsg = msg
		}
		cc.client, _ = rpchttp.New(endpoint.Url, "/websocket")
		status, err := cc.client.Status(ctx)
		if err != nil {
			msg := fmt.Sprintf("❌ could not start client for %s: (%s) %s", cc.name, endpoint.Url, err)
			down(msg)
			l(msg)
			continue
		}
		if status.NodeInfo.Network != cc.ChainId {
			msg := fmt.Sprintf("chain id %s on %s does not match, expected %s, skipping", status.NodeInfo.Network, endpoint.Url, cc.ChainId)
			down(msg)
			l(msg)
			continue
		}
		if status.SyncInfo.CatchingUp {
			msg := fmt.Sprint("🐢 node is not synced, skipping ", endpoint.Url)
			endpoint.syncing = true
			down(msg)
			l(msg)
			continue
		}
		cc.noNodes = false
		return nil
	}
	cc.noNodes = true
	cc.lastError = "no usable RPC endpoints available for " + cc.ChainId
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
	return errors.New("📵 no usable endpoints available for " + cc.ChainId)
}

func (cc *ChainConfig) monitorHealth(ctx context.Context, chainName string) {
	tick := time.NewTicker(time.Minute)
	if cc.client == nil {
		e := cc.newRpc()
		if e != nil {
			l("💥", cc.ChainId, e)
		}
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
						td.statsChan <- cc.mkUpdate(metricNodeDownSeconds, time.Now().Sub(node.downSince).Seconds(), node.Url)
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
			cc.lastValInfo = cc.valInfo // FIXME: this isn't how you deep copy *struct
			err = cc.GetValInfo(false)
			if err != nil {
				l("❓ refreshing signing info for", cc.ValAddress, err)
			}
		}
	}
}
