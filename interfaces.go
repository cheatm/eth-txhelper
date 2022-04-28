package txhelper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type TxSigner interface {
	Sign(*types.Transaction, *big.Int) (*types.Transaction, error)
	Account() common.Address
}
