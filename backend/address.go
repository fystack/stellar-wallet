package main

import (
	"encoding/base32"

	"github.com/mr-tron/base58"
)

// deriveAddress turns a 32-byte ed25519 public key into a chain address.
func deriveAddress(chain string, pub []byte) string {
	switch chain {
	case "solana":
		// Solana addresses are the base58 of the raw ed25519 public key.
		return base58.Encode(pub)
	case "stellar":
		return stellarStrkey(pub)
	default:
		return base58.Encode(pub)
	}
}

// stellarStrkey encodes an ed25519 public key as a Stellar account address (G...).
// Format: version byte (6<<3) || 32-byte key || CRC16-XModem checksum, base32 (no pad).
func stellarStrkey(pub []byte) string {
	const versionAccountID = 6 << 3 // -> 'G'
	payload := append([]byte{versionAccountID}, pub...)
	checksum := crc16XModem(payload)
	payload = append(payload, checksum[0], checksum[1])
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payload)
}

// crc16XModem returns the 2-byte little-endian CRC16 (XModem) checksum.
func crc16XModem(data []byte) [2]byte {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return [2]byte{byte(crc), byte(crc >> 8)} // little-endian
}
