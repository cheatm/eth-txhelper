package sender_test

import (
	"context"
	"encoding/json"
	txhelper "eth-txhelper"
	"eth-txhelper/sender"
	"log"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const Erc20ABI = "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol_\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

var ENDPOINT = os.Getenv("ETH_ENDPOINT")
var KEYSTORE = os.Getenv("KEYSTORE")
var ACCOUNT = os.Getenv("ACCOUNT")
var PASSWORD = os.Getenv("PASSWORD")
var TOKEN0 = os.Getenv("TOKEN0")
var TOKEN1 = os.Getenv("TOKEN1")

var account common.Address
var client *ethclient.Client
var signer txhelper.TxSigner
var chainId *big.Int
var erc20abi *abi.ABI
var acp sender.AsyncControlParams

func HandlError(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	var err error
	account = common.HexToAddress(ACCOUNT)
	signer, err = sender.LocalSignerFromKs(KEYSTORE, ACCOUNT, PASSWORD)
	HandlError(err)
	client, err = ethclient.Dial(ENDPOINT)
	HandlError(err)
	chainId, err = client.ChainID(context.Background())
	HandlError(err)
	parsed, err := abi.JSON(strings.NewReader(Erc20ABI))
	HandlError(err)
	erc20abi = &parsed
	acp = sender.AsyncControlParams{
		PollingSecs:       1,
		Retry:             5,
		ExpectedTimeout:   60,
		UnexpectedTimeout: 60,
	}
}

func TestInit(t *testing.T) {
	log.Println("Init")
}

func TestEnv(t *testing.T) {
	log.Println(ENDPOINT)
	log.Println(ACCOUNT)
}

func TestBalance(t *testing.T) {
	balance, err := client.BalanceAt(context.Background(), account, nil)
	HandlError(err)
	log.Printf("Balance of %s: %d", account, balance)
}

func TestNonce(t *testing.T) {
	nonceAt, err := client.NonceAt(context.Background(), account, nil)
	HandlError(err)
	log.Println(nonceAt)
}

func TestSelfTransfer(t *testing.T) {
	msg := ethereum.CallMsg{From: account, To: (*common.Address)(account[:])}
	tx, err := sender.BuildTransferTx(client, msg)
	HandlError(err)
	ShowJson(tx)
	signedTx, err := signer.Sign(tx, chainId)
	HandlError(err)
	log.Println(signedTx.Hash())
	err = client.SendTransaction(context.Background(), signedTx)
	HandlError(err)
	nonceAt, mined := sender.WaitPendingNonce(client, account, signedTx.Nonce(), 60, 1)
	if mined {
		log.Printf("mined: %d", signedTx.Nonce())
		log.Printf("nonce: %d", nonceAt)
		rc, err := client.TransactionReceipt(context.Background(), signedTx.Hash())
		HandlError(err)
		ShowJson(rc)

	} else {
		log.Printf("timeout: %d, %d", signedTx.Nonce(), nonceAt)
		TestNonce(t)
	}
}

func TestErc20Transfer(t *testing.T) {
	token0 := common.HexToAddress(TOKEN0)
	msg := ethereum.CallMsg{From: account, To: (*common.Address)(token0[:])}
	mtd := sender.ContractMethodParams{
		Abi:    erc20abi,
		Method: "transfer",
		Params: []interface{}{account, big.NewInt(0)},
	}
	trx, err := sender.BuildCallContractTx(client, msg, mtd)
	HandlError(err)
	ShowJson(trx)
	signedTx, rc, err := sender.TransactWithReceipt(acp, client, chainId, signer, trx)
	HandlError(err)
	ShowJson(signedTx)
	ShowJson(rc)
}

type Erc20 struct {
	sender.ContractTransactor
}

func (erc20 *Erc20) Transfer(
	acp sender.AsyncControlParams,
	address common.Address, amount *big.Int,
) (
	*types.Transaction,
	*types.Receipt,
	error,
) {
	return erc20.TransactWithReceipt(acp, "transfer", []interface{}{address, amount})
}

func TestTransactor(t *testing.T) {
	transactor, err := sender.NewContractTransactor(client, erc20abi, common.HexToAddress(TOKEN0), signer)
	HandlError(err)
	erc20 := Erc20{*transactor}
	tx, rc, err := erc20.Transfer(acp, account, big.NewInt(0))
	HandlError(err)
	log.Println(tx.Hash())
	log.Println(rc.Status)
}

func ShowJson(obj json.Marshaler) {
	data, err := obj.MarshalJSON()
	if err != nil {
		log.Println("Marshall failed:", err)
	}
	log.Println(string(data))
}
