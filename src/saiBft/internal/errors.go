package internal

import "errors"

var (
	errNoBlocks              = errors.New("No blocks found")
	errNoConnectedNodesFound = errors.New("No connected nodes found")
)
