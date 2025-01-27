package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	firstBlockTimes = struct {
		sync.Mutex
		Times map[string]time.Time
	}{Times: make(map[string]time.Time)}
)

type NodeStatistics struct {
	URL                 string
	TotalDelay          time.Duration
	TotalBlocksReceived int                 // Total number of blocks received by this node
	ProcessedBlocks     map[string]struct{} // Set of blocks that have been processed
	mutex               sync.Mutex
}

func (ns *NodeStatistics) UpdateDelay(blockHash string, blockNumber uint64, receivedTime time.Time) {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	// Increment the total blocks received
	ns.TotalBlocksReceived++

	// Check if the block has already been processed for this node
	if _, exists := ns.ProcessedBlocks[blockHash]; exists {
		return // Block already processed, ignore it
	}
	ns.ProcessedBlocks[blockHash] = struct{}{}

	firstBlockTimes.Lock()
	firstTime, exists := firstBlockTimes.Times[blockHash]
	if !exists {
		firstBlockTimes.Times[blockHash] = receivedTime
		firstBlockTimes.Unlock()
		fmt.Printf("Node: %s, Block: %s [%d], First receive, Total Blocks: %d\n", ns.URL, blockHash, blockNumber, ns.TotalBlocksReceived)
	} else {
		firstBlockTimes.Unlock()
		delay := receivedTime.Sub(firstTime)
		ns.TotalDelay += delay
		fmt.Printf("Node: %s, Block: %s [%d], Delay: %s, Total Delay: %s, Total Blocks: %d\n", ns.URL, blockHash, blockNumber, delay, ns.TotalDelay, ns.TotalBlocksReceived)
	}
}

func subscribeToNode(url string, stats *NodeStatistics, headersChan chan *types.Header, errChan chan error) {
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
			receivedTime := time.Now()
			blockHash := header.Hash().Hex()
			blockNumber := header.Number.Uint64()
			stats.UpdateDelay(blockHash, blockNumber, receivedTime)
		}
	}
}

func main() {
	nodes := []string{
		"wss://base.gateway.tenderly.co",
		"wss://base-rpc.publicnode.com",
		"wss://base-mainnet.g.alchemy.com/v2/K1SeWBita0JiyCpo9JaydfB9z-nmG87x",
	}

	headersChan := make(chan *types.Header)
	errChan := make(chan error)
	statsMap := make(map[string]*NodeStatistics)

	for _, nodeURL := range nodes {
		stats := &NodeStatistics{
			URL:             nodeURL,
			ProcessedBlocks: make(map[string]struct{}),
		}
		statsMap[nodeURL] = stats
		go subscribeToNode(nodeURL, stats, headersChan, errChan)
	}

	for {
		select {
		case err := <-errChan:
			log.Fatal(err)
		}
	}
}
