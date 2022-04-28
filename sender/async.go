package sender

import (
	"context"
	txhelper "eth-txhelper"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	NotFoundMessage    = "not found"
	UnderpricedMessage = "replacement transaction underpriced"
	NonceTooLowMessage = "nonce too low"
	AlreadKnownMessage = "already known"
)

type AsyncControlParams struct {
	PollingSecs       int64 // Seconds to wait when polling
	Retry             int64 // Max retry times when errors occur
	ExpectedTimeout   int64 // Seconds to wait for an expected transaction to be mined
	UnexpectedTimeout int64 // Seconds to wait for an unexpected transaction to be mined
	NonceImmutable    bool  // Set the nonce of origin tx to be immutable
}

func RaiseGasPrice(price *big.Int) *big.Int {
	gasPrice := big.NewInt(0)
	gasPrice.Mul(
		price,
		big.NewInt(110),
	)
	gasPrice.Div(gasPrice, big.NewInt(100))
	return gasPrice
}

func RaiseTxGas(client *ethclient.Client, basePrice *big.Int) *big.Int {
	suggestGasPrice, sgpErr := client.SuggestGasPrice(context.Background())
	// suggestGasPrice, sgpErr := big.NewInt(0), error(nil)
	gasPrice := RaiseGasPrice(basePrice)
	if sgpErr == nil {
		if gasPrice.Cmp(suggestGasPrice) < 0 {
			gasPrice.Set(suggestGasPrice)
		}
	}
	return gasPrice
}

func WaitPendingNonce(client *ethclient.Client, account common.Address, nonce uint64, timeout int64, pollingSecs int64) (uint64, bool) {
	sleepDuration := time.Second * time.Duration(pollingSecs)
	expireAt := time.Now().Unix() + timeout
	var nonceAt uint64
	var err error
	for time.Now().Unix() < expireAt {
		nonceAt, err = client.NonceAt(context.Background(), account, nil)
		if err == nil {
			if nonceAt > nonce {
				return nonceAt, true
			}
		}
		time.Sleep(sleepDuration)
	}
	return nonceAt, false

}

func TransactWithReceipt(
	param AsyncControlParams,
	client *ethclient.Client,
	chainId *big.Int,
	signer txhelper.TxSigner,
	tx *types.Transaction,
) (*types.Transaction, *types.Receipt, error) {

SIGN:
	signedTx, err := signer.Sign(tx, chainId)
	if err != nil {
		return nil, nil, err
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		if param.Retry <= 0 {
			return nil, nil, err
		}
		switch err.Error() {
		case UnderpricedMessage:
			// nonce already taken by others, wait for it to be mined.
			if nonceAt, filled := WaitPendingNonce(client, signer.Account(), tx.Nonce(), param.UnexpectedTimeout, param.PollingSecs); filled {
				// if tx mined in time, send tx with new nonce and gasPrice
				if param.NonceImmutable {
					return nil, nil, fmt.Errorf("nonce immutable: %d", tx.Nonce())
				}
				gasPrice := RaiseTxGas(client, tx.GasPrice())
				tx = types.NewTransaction(nonceAt, *tx.To(), tx.Value(), tx.Gas(), gasPrice, tx.Data())
				param.Retry--
				goto SIGN
			} else {
				// if tx not mined in time, replace it with higher gasPrice
				gasPrice := RaiseTxGas(client, tx.GasPrice())
				tx = types.NewTransaction(tx.Nonce(), *tx.To(), tx.Value(), tx.Gas(), gasPrice, tx.Data())
				param.Retry--
				goto SIGN
			}
		case NonceTooLowMessage:
			if param.NonceImmutable {
				return nil, nil, fmt.Errorf("nonce immutable: %d", tx.Nonce())
			}
			var nonceAt uint64
			for {
				nonceAt, err = client.NonceAt(context.Background(), signer.Account(), nil)
				if err != nil {
					if param.Retry > 0 {
						param.Retry--
					} else {
						return nil, nil, err
					}
				} else {
					break
				}
			}
			gasPrice := RaiseTxGas(client, tx.GasPrice())
			tx = types.NewTransaction(nonceAt, *tx.To(), tx.Value(), tx.Gas(), gasPrice, tx.Data())
			param.Retry--
			goto SIGN
		case AlreadKnownMessage:
			goto WAIT
		}
		return nil, nil, err

	}

WAIT:

	// Wait for signedTx to be mined
	if nonceAt, filled := WaitPendingNonce(client, signer.Account(), signedTx.Nonce(), param.ExpectedTimeout, param.PollingSecs); filled {
		// if tx at nonce is mined, check receipt
		rc, err := client.TransactionReceipt(context.Background(), signedTx.Hash())
		if err != nil {
			if param.Retry <= 0 {
				return signedTx, nil, err
			}
			switch err.Error() {
			case NotFoundMessage:
				// tx replaced by others
				gasPrice := RaiseTxGas(client, tx.GasPrice())
				tx = types.NewTransaction(nonceAt, *signedTx.To(), signedTx.Value(), signedTx.Gas(), gasPrice, tx.Data())
				param.Retry--
				goto SIGN
			}
		}
		return signedTx, rc, err
	} else {
		//
		if param.Retry > 0 {

			gasPrice := RaiseTxGas(client, tx.GasPrice())
			tx = types.NewTransaction(tx.Nonce(), *tx.To(), tx.Value(), tx.Gas(), gasPrice, tx.Data())
			param.Retry--
			goto SIGN
		} else {
			return signedTx, nil, err
		}
	}
}
