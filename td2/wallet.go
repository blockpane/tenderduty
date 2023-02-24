package tenderduty

import (
	"context"
	"errors"
	"fmt"

	banking "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (cc *ChainConfig) getWalletInfo(ctx context.Context, wallet *WalletConfig) (err error) {
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
	return
}
