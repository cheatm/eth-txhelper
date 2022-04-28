package sender

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type LocalSigner struct {
	key *keystore.Key
}

func NewLocalSigner(key *keystore.Key) *LocalSigner {
	return &LocalSigner{key}
}

func LocalSignerFromKs(path string, hexaddr string, password string) (*LocalSigner, error) {
	address := common.HexToAddress(hexaddr)
	account := accounts.Account{Address: address}
	ks := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	keybytes, err := ks.Export(account, password, "")
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keybytes, "")
	if err != nil {
		return nil, err
	}
	return &LocalSigner{key}, nil
}

func (signer *LocalSigner) Sign(tx *types.Transaction, chainId *big.Int) (*types.Transaction, error) {
	return types.SignTx(tx, types.NewEIP155Signer(chainId), signer.key.PrivateKey)
}

func (signer *LocalSigner) Account() common.Address {
	return signer.key.Address
}
