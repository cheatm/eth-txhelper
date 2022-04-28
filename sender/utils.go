package sender

import (
	"context"
	txhelper "eth-txhelper"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TxParams struct {
	From     common.Address
	To       common.Address
	Value    *big.Int
	GasPrice *big.Int
	Gas      uint64
	Nonce    uint64
	Data     []byte
}

type ContractMethodParams struct {
	Abi    *abi.ABI
	Method string
	Params []interface{}
}

func BuildTransferTx(client *ethclient.Client, callMsg ethereum.CallMsg) (*types.Transaction, error) {
	var err error
	ctx := context.Background()

	nonce, err := client.NonceAt(ctx, callMsg.From, nil)
	if err != nil {
		return nil, err
	}
	if callMsg.GasPrice == nil {
		callMsg.GasPrice, err = client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
	}

	if callMsg.Gas == 0 {
		if len(callMsg.Data) == 0 {
			callMsg.Gas = 21000
		} else {
			callMsg.Gas, err = client.EstimateGas(ctx, callMsg)
			if err != nil {
				return nil, err
			}
		}
	}

	return types.NewTransaction(nonce, *callMsg.To, callMsg.Value, callMsg.Gas, callMsg.GasPrice, callMsg.Data), nil

}

func BuildCallContractTx(client *ethclient.Client, callMsg ethereum.CallMsg, methodParams ContractMethodParams) (*types.Transaction, error) {
	data, err := methodParams.Abi.Pack(methodParams.Method, methodParams.Params...)
	if err != nil {
		return nil, err
	}
	callMsg.Data = data
	return BuildTransferTx(client, callMsg)
}

type ContractTransactor struct {
	client  *ethclient.Client
	chainId *big.Int
	abi     *abi.ABI
	address *common.Address
	signer  txhelper.TxSigner
}

func NewContractTransactor(
	client *ethclient.Client,
	abi *abi.ABI,
	address common.Address,
	signer txhelper.TxSigner,
) (*ContractTransactor, error) {
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	return &ContractTransactor{
		client:  client,
		chainId: chainId,
		abi:     abi,
		address: (*common.Address)(address[:]),
		signer:  signer,
	}, nil
}

func (transactor *ContractTransactor) BuildTrx(method string, params []interface{}) (*types.Transaction, error) {
	return BuildCallContractTx(
		transactor.client,
		ethereum.CallMsg{
			From: transactor.signer.Account(),
			To:   transactor.address,
		},
		ContractMethodParams{
			Abi:    transactor.abi,
			Method: method,
			Params: params,
		},
	)
}

func (transactor *ContractTransactor) TransactWithReceipt(acp AsyncControlParams, method string, params []interface{}) (
	*types.Transaction,
	*types.Receipt,
	error,
) {
	tx, err := transactor.BuildTrx(method, params)
	if err != nil {
		return nil, nil, err
	}
	return TransactWithReceipt(
		acp, transactor.client, transactor.chainId, transactor.signer, tx,
	)

}
