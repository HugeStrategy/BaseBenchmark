package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"time"
)

// NodeInfo stores information about a blockchain node subscription
type NodeInfo struct {
	URL         string
	LatestBlock *types.Header
	ReceiveTime time.Time
}

func subscribeToNode(url string, headersChan chan *types.Header, errChan chan error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		errChan <- err
		return
	}

	sub, err := client.SubscribeNewHead(context.Background(), headersChan)
	if err != nil {
		errChan <- err
		return
	}

	for {
		select {
		case err := <-sub.Err():
			errChan <- err
			return
		case header := <-headersChan:
			fmt.Printf("New block from %s: %s at %v\n", url, header.Hash().Hex(), time.Now())
		}
	}
}

func main() {
	nodes := []string{
		"wss://ropsten.infura.io/ws",
	}

	headersChan := make(chan *types.Header)
	errChan := make(chan error)

	for _, nodeURL := range nodes {
		go subscribeToNode(nodeURL, headersChan, errChan)
	}

	for {
		select {
		case err := <-errChan:
			log.Fatal(err)
		}
	}
}
