package tenderduty

import (
	"context"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"strings"
	"time"
)

// ValInfo holds most of the stats/info used for secondary alarms. It is refreshed roughly every minute.
type ValInfo struct {
	Moniker    string `json:"moniker"`
	Bonded     bool   `json:"bonded"`
	Jailed     bool   `json:"jailed"`
	Tombstoned bool   `json:"tombstoned"`
	Missed     int64  `json:"missed"`
	Window     int64  `json:"window"`
	Conspub    []byte `json:"conspub"`
	Valcons    string `json:"valcons"`
}

// GetValInfo the first bool is used to determine if extra information about the validator should be printed.
func (cc *ChainConfig) GetValInfo(first bool) (err error) {
	if cc.client == nil {
		return errors.New("nil rpc client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if cc.valInfo == nil {
		cc.valInfo = &ValInfo{}
	}

	// Fetch info from /cosmos.staking.v1beta1.Query/Validator
	// it's easier to ask people to provide valoper since it's readily available on
	// explorers, so make it easy and lookup the consensus key for them.
	cc.valInfo.Conspub, cc.valInfo.Moniker, cc.valInfo.Jailed, cc.valInfo.Bonded, err = getVal(ctx, cc.client, cc.ValAddress)
	if err != nil {
		return
	}
	if first && cc.valInfo.Bonded {
		l(fmt.Sprintf("⚙️ found %s (%s) in validator set", cc.ValAddress, cc.valInfo.Moniker))
	} else if first && !cc.valInfo.Bonded {
		l(fmt.Sprintf("❌ %s (%s) is INACTIVE", cc.ValAddress, cc.valInfo.Moniker))
	}

	// need to know the prefix for when we serialize the slashing info query, this is too fragile.
	// for now, we perform specific chain overrides based on known values because the valoper is used
	// in so many places.
	var prefix string
	split := strings.Split(cc.ValAddress, "valoper")
	if len(split) != 2 {
		if pre, ok := altValopers.getAltPrefix(cc.ValAddress); ok {
			cc.valInfo.Valcons, err = bech32.ConvertAndEncode(pre, cc.valInfo.Conspub[:20])
			if err != nil {
				return
			}
		} else {
			err = errors.New("❓ could not determine bech32 prefix from valoper address: " + cc.ValAddress)
			return
		}
	} else {
		prefix = split[0] + "valcons"
		cc.valInfo.Valcons, err = bech32.ConvertAndEncode(prefix, cc.valInfo.Conspub[:20])
		if err != nil {
			return
		}
	}
	if first {
		l("⚙️", cc.ValAddress[:20], "... is using consensus key:", cc.valInfo.Valcons)
	}

	// get current signing information (tombstoned, missed block count)
	qSigning := slashing.QuerySigningInfoRequest{ConsAddress: cc.valInfo.Valcons}
	b, err := qSigning.Marshal()
	if err != nil {
		return
	}
	resp, err := cc.client.ABCIQuery(ctx, "/cosmos.slashing.v1beta1.Query/SigningInfo", b)
	if resp == nil || resp.Response.Value == nil {
		err = errors.New("could not query validator slashing status, got empty response")
		return
	}
	slash := &slashing.QuerySigningInfoResponse{}
	err = slash.Unmarshal(resp.Response.Value)
	if err != nil {
		return
	}
	cc.valInfo.Tombstoned = slash.ValSigningInfo.Tombstoned
	if cc.valInfo.Tombstoned {
		l(fmt.Sprintf("❗️☠️ %s (%s) is tombstoned 🪦❗️", cc.ValAddress, cc.valInfo.Moniker))
	}
	cc.valInfo.Missed = slash.ValSigningInfo.MissedBlocksCounter
	if td.Prom {
		td.statsChan <- cc.mkUpdate(metricWindowMissed, float64(cc.valInfo.Missed), "")
	}

	// finally get the signed blocks window
	if cc.valInfo.Window == 0 {
		qParams := &slashing.QueryParamsRequest{}
		b, err = qParams.Marshal()
		if err != nil {
			return
		}
		resp, err = cc.client.ABCIQuery(ctx, "/cosmos.slashing.v1beta1.Query/Params", b)
		if err != nil {
			return
		}
		if resp.Response.Value == nil {
			err = errors.New("🛑 could not query slashing params, got empty response")
			return
		}
		params := &slashing.QueryParamsResponse{}
		err = params.Unmarshal(resp.Response.Value)
		if err != nil {
			return
		}
		if first && td.Prom {
			td.statsChan <- cc.mkUpdate(metricWindowSize, float64(params.Params.SignedBlocksWindow), "")
			td.statsChan <- cc.mkUpdate(metricTotalNodes, float64(len(cc.Nodes)), "")
		}
		cc.valInfo.Window = params.Params.SignedBlocksWindow
	}
	return
}

// getVal returns the public key, moniker, and if the validator is jailed.
func getVal(ctx context.Context, client *rpchttp.HTTP, valoper string) (pub []byte, moniker string, jailed, bonded bool, err error) {
	q := staking.QueryValidatorRequest{
		ValidatorAddr: valoper,
	}
	b, err := q.Marshal()
	if err != nil {
		return
	}
	resp, err := client.ABCIQuery(ctx, "/cosmos.staking.v1beta1.Query/Validator", b)
	if err != nil {
		return
	}
	if resp.Response.Value == nil {
		return nil, "", false, false, errors.New("could not find validator " + valoper)
	}
	val := &staking.QueryValidatorResponse{}
	err = val.Unmarshal(resp.Response.Value)
	if err != nil {
		return
	}
	pk := ed25519.PubKey{}
	err = pk.Unmarshal(val.Validator.ConsensusPubkey.Value)
	if err != nil {
		return
	}
	//fmt.Println(pk.Address().Bytes())
	return pk.Address().Bytes(), val.Validator.GetMoniker(), val.Validator.Jailed, val.Validator.Status == 3, nil
}
