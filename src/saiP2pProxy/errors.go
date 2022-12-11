package main

import "errors"

var (
	errNoBlocks              = errors.New("No blocks found")
	errConnectedAddrNotFound = errors.New("p2p response address not found in p2p connected list")
	errWrongMsgType          = errors.New("wrong message type provided")
)
