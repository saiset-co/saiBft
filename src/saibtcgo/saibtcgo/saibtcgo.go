package saibtcgo

import (
	"saibtcgo/saibtcgo/btc"
	"bytes"
	"encoding/hex"
	"encoding/base64"
	"flag"
	"errors"
	"fmt"
	"crypto/rand"
)

var (
	Testnet      = false
	Uncompressed = false
)

type BtcKey struct {
	Private string
	Priv    []byte
	Public  string
	Pub     []byte
	Address string
}

func (b *BtcKey) Dump() {
	fmt.Println(b.Private)
	fmt.Println(b.Priv)
	fmt.Println(b.Public)
	fmt.Println(b.Pub)
	fmt.Println(b.Address)
}

func ver_pubkey() byte {
	return btc.AddrVerPubkey(Testnet)
}

func ver_secret() byte {
	return ver_pubkey() + 0x80
}

func GenerateKeyPair() (*BtcKey, error) {
	b := make([]byte, 256)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	seedKey := make([]byte, 32)
	btc.ShaHash(b, b)
	rand.Read(seedKey[:])
	prvKey := make([]byte, 32)
	btc.ShaHash(seedKey, prvKey)
	copy(seedKey, prvKey)
	rand.Read(seedKey[:])

	rec := btc.NewPrivateAddr(prvKey, ver_secret(), !Uncompressed)

	return &BtcKey{
		rec.String(),
		rec.Key,
		hex.EncodeToString(rec.BtcAddr.Pubkey),
		rec.BtcAddr.Pubkey,
		rec.BtcAddr.String(),
	}, nil
}

func SignMessage(message string, privatekey string) (string, error) {
	var hash []byte
	var signkey *btc.PrivateAddr

	signkey, err := btc.DecodePrivateAddr(privatekey)
	if err != nil {
		return "", err
	}

	var msg []byte
	msg = []byte(message)

	hash = make([]byte, 32)
	btc.HashFromMessage(msg, hash)

	btcsig := new(btc.Signature)
	var sb [65]byte
	sb[0] = 27
	if signkey.IsCompressed() {
		sb[0] += 4
	}

	r, s, err := btc.EcdsaSign(signkey.Key, hash)
	if err != nil {
		return "", err
	}

	btcsig.R.Set(r)
	btcsig.S.Set(s)

	rd := btcsig.R.Bytes()
	sd := btcsig.S.Bytes()
	copy(sb[1+32-len(rd):], rd)
	copy(sb[1+64-len(sd):], sd)

	rpk := btcsig.RecoverPublicKey(hash[:], 0)
	sa := btc.NewAddrFromPubkey(rpk.Bytes(signkey.IsCompressed()), signkey.BtcAddr.Version)
	if sa.Hash160 == signkey.BtcAddr.Hash160 {
		return base64.StdEncoding.EncodeToString(sb[:]), nil
	}

	rpk = btcsig.RecoverPublicKey(hash[:], 1)
	sa = btc.NewAddrFromPubkey(rpk.Bytes(signkey.IsCompressed()), signkey.BtcAddr.Version)
	if sa.Hash160 == signkey.BtcAddr.Hash160 {
		sb[0]++
		return base64.StdEncoding.EncodeToString(sb[:]), nil
	}
	return "", errors.New("Something went wrong. The message has not been signed.")
}

func VerifySignature(message string, signature string, address string) (bool, error) {
	ad, err := btc.NewAddrFromString(address)
	if err != nil {
		flag.PrintDefaults()
		return false, err
	}

	nv, btcsig, err := btc.ParseMessageSignature(signature)
	if err != nil {
		return false, err
	}

	var msg []byte
	if message != "" {
		msg = []byte(message)
	}

	hash := make([]byte, 32)
	btc.HashFromMessage(msg, hash)

	compressed := false
	if nv >= 31 {
		nv -= 4
		compressed = true
	}

	pub := btcsig.RecoverPublicKey(hash[:], int(nv-27))
	if pub != nil {
		pk := pub.Bytes(compressed)
		ok := btc.EcdsaVerify(pk, btcsig.Bytes(), hash)
		if ok {
			sa := btc.NewAddrFromPubkey(pk, ad.Version)
			if ad.Hash160 != sa.Hash160 {
				if bytes.IndexByte(msg, '\r') != -1 {
					return false, errors.New("You have CR chars in the message. Try to verify with -u switch.")
				}
				return false, errors.New("BAD signature for " + ad.String())
			} else {
				return true, nil
			}
		} else {
			return false, errors.New("BAD signature")
		}
	} else {
		return false, errors.New("BAD, BAD, BAD signature")
	}
}
