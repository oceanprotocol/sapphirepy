package main

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -L. -lsapphirewrapper
// #include "sapphirewrapper.h"
// #include <stdlib.h>

import (
	"C"
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	sapphire "github.com/oasisprotocol/sapphire-paratime/clients/go"
)
import (
	"encoding/hex"
	"fmt"
	"os"
)

//export SendETHTransaction
func SendETHTransaction(keyHexC *C.char, myAddrC *C.char, toAddrC *C.char, rpcUrl *C.char, valueC C.int, gasLimitC C.int, dataC *C.char, gasCostGweiC C.int, nonceC C.int) (C.int, *C.char) {
	keyhex := C.GoString(keyHexC)
	myAddrStr := C.GoString(myAddrC)
	toAddrStr := C.GoString(toAddrC)
	rpcUrlStr := C.GoString(rpcUrl)
	gasLimit := uint64(gasLimitC)
	datahex := C.GoString(dataC)
	gasCostGwei := uint64(gasCostGweiC)
	nonce := uint64(nonceC)
	var gasPrice *big.Int
	logLevel := os.Getenv("LOGLEVEL")

	value := big.NewInt(int64(valueC))
	value = value.Mul(value, big.NewInt(1000000000)) // convert gwei to wei

	c, err := ethclient.Dial(rpcUrlStr)
	if err != nil {
		return -1, nil
	}

	key, err := crypto.HexToECDSA(keyhex)
	if err != nil {
		return -2, nil
	}

	myAddr := common.HexToAddress(myAddrStr)
	toAddr := common.HexToAddress(toAddrStr)
	if nonce == 0 {
		nonce, err = c.PendingNonceAt(context.Background(), myAddr)
		fmt.Println("Pending nonce:", nonce)
		if err != nil {
			return -3, nil
		}
	}

	chainId, err := c.NetworkID(context.Background())
	if err != nil {
		return -4, nil
	}

	var data []byte
	var encryptedData []byte
	if datahex == "" {
		data = nil
		encryptedData = nil
		if logLevel == "DEBUG" {
			fmt.Println("Transaction data is empty")
		}
	} else {
		// remove 0x if it exists
		if datahex[:2] == "0x" {
			datahex = datahex[2:]
		}
		if logLevel == "DEBUG" {
			fmt.Println("Transaction data hex:", datahex)
		}
		data, err = hex.DecodeString(datahex)
		if err != nil {
			return -42, nil
		}
		if logLevel == "DEBUG" {
			fmt.Println("Decoded data:", data)
		}
		cipher, err := sapphire.NewCipher(chainId.Uint64())
		if err != nil {
			fmt.Println("Error creating cipher:", err)
			return 99, nil
		}
		encryptedData = cipher.EncryptEncode(data)
		if logLevel == "DEBUG" {
			fmt.Println("Encrypted data:", encryptedData)
		}
	}

	if gasCostGwei == 0 {
		gasPrice, err = c.SuggestGasPrice(context.Background())
		fmt.Println("SuggestGasPrice:", gasPrice.String())
		if err != nil {
			return -43, nil
		}
	} else {
		gasPrice = big.NewInt(int64(gasCostGwei))
		gasPrice = gasPrice.Mul(gasPrice, big.NewInt(1000000000)) // convert gwei to wei
	}

	tx := types.NewTx(
		&types.LegacyTx{
			Nonce:    nonce,
			To:       &toAddr,
			Value:    value,
			Data:     encryptedData,
			Gas:      gasLimit,
			GasPrice: gasPrice,
		},
	)

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainId), key)
	if err != nil {
		return -5, nil
	}

	err = c.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Println(err)
		return -6, nil
	}

	return 0, C.CString(signedTx.Hash().Hex())
}

func main() {}
