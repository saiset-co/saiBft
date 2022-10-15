package main

import (
	"fmt"
	"../saibtcgo/saibtcgo"
)

func main() {
	mess := "This is an example of a signed message."

	btcKey, err := saibtcgo.GenerateKeyPair()
	if err != nil {
		fmt.Println(err)
		return
	}
	btcKey.Dump()

	signature, err := saibtcgo.SignMessage(mess, btcKey.Private)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(signature)
	valid, err := saibtcgo.VerifySignature(mess, "H9PPoeUtFOli2NwDDgPz2IMRbEhyZ5ngbRrRhsgeOq83CeMmH7tmXCHUmuX6rj0THQPjsMd2K6mBQl6XL8gdAAM=", "1MRBqNJZ5eBQw531YYFYCtp86TMcQQRzYN")
	// valid, err := saibtcgo.VerifySignature(mess, signature, btcKey.Address)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(valid)
}
