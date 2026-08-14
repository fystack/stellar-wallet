package chain

import (
	"encoding/base32"

	"github.com/mr-tron/base58"
)

func DeriveAddress(chainName string, publicKey []byte) string {
	switch chainName {
	case "solana":
		return base58.Encode(publicKey)
	case "stellar":
		return stellarStrkey(publicKey)
	default:
		return base58.Encode(publicKey)
	}
}

func stellarStrkey(publicKey []byte) string {
	const versionAccountID = 6 << 3
	payload := append([]byte{versionAccountID}, publicKey...)
	checksum := crc16XModem(payload)
	payload = append(payload, checksum[0], checksum[1])
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payload)
}

func crc16XModem(data []byte) [2]byte {
	var crc uint16
	for _, value := range data {
		crc ^= uint16(value) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return [2]byte{byte(crc), byte(crc >> 8)}
}

func ExplorerAddress(chainName, address string) string {
	switch chainName {
	case "stellar":
		return "https://stellar.expert/explorer/testnet/account/" + address
	case "solana":
		return "https://explorer.solana.com/address/" + address + "?cluster=devnet"
	default:
		return ""
	}
}
