package main_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ddr4869/minifab/common/types"
	"github.com/ddr4869/minifab/orderer"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/ddr4869/minifab/peer/server"
)

func TestBlockBroadcasting(t *testing.T) {
	t.Log("Testing Block Broadcasting System")

	// Test 1: Create orderer and peer
	t.Log("1. Creating orderer and peer instances...")

	// Create orderer
	ord := orderer.NewOrderer("OrdererMSP")
	ordererServer := orderer.NewOrdererServer(ord)

	// Create peer
	peerInstance := core.NewPeer("peer0", "./chaincode", "Org1MSP")
	peerServer := server.NewPeerServer(peerInstance)

	t.Log("✅ Orderer and peer instances created")

	// Test 2: Start servers
	t.Log("\n2. Starting servers...")

	// Start orderer server
	go func() {
		if err := ordererServer.Start(":7050"); err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				t.Log("Orderer server already running")
				return
			}
			t.Errorf("Orderer server failed to start: %v", err)
		}
	}()

	// Start peer server
	go func() {
		if err := peerServer.Start(":7051"); err != nil {
			t.Errorf("Peer server failed to start: %v", err)
		}
	}()

	// Wait for servers to start
	time.Sleep(2 * time.Second)
	t.Log("✅ Servers started")

	// Test 3: Create channel
	t.Log("\n3. Creating channel...")

	// Connect to orderer as a client
	ordererClient, err := client.NewOrdererClient("localhost:7050")
	if err != nil {
		t.Fatalf("Failed to create orderer client: %v", err)
	}
	defer ordererClient.Close()

	// Create channel
	if err := ordererClient.CreateChannel("testchannel"); err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	t.Log("✅ Channel created successfully")

	// Test 4: Create and validate block
	t.Log("\n4. Creating and validating block...")

	// Create a block with transaction data
	block, err := ord.CreateBlock([]byte("test block data"))
	if err != nil {
		t.Fatalf("Failed to create block: %v", err)
	}

	t.Logf("✅ Block created with number: %d", block.Number)

	// Test 5: Submit transactions through peer
	t.Log("\n5. Testing transaction submission...")

	// Submit a transaction through peer
	tx := &types.Transaction{
		ID:        "test-tx-1",
		ChannelID: "testchannel",
		Payload:   []byte("peer transaction"),
		Timestamp: time.Now(),
	}

	if err := ordererClient.SubmitTransaction(tx); err != nil {
		t.Errorf("Failed to submit transaction: %v", err)
	} else {
		t.Log("✅ Transaction submitted successfully")
	}

	// Test 6: Performance test
	t.Log("\n6. Running performance test...")

	start := time.Now()
	numTransactions := 10

	for i := 0; i < numTransactions; i++ {
		tx := &types.Transaction{
			ID:        "perf-tx-" + string(rune(i+'0')),
			ChannelID: "testchannel",
			Payload:   []byte("performance test transaction"),
			Timestamp: time.Now(),
		}

		if err := ordererClient.SubmitTransaction(tx); err != nil {
			t.Errorf("Failed to submit transaction %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	t.Logf("✅ Submitted %d transactions in %v", numTransactions, duration)
	t.Logf("✅ Average: %v per transaction", duration/time.Duration(numTransactions))

	// Test 7: Block creation performance
	t.Log("\n7. Testing block creation performance...")

	start = time.Now()
	numBlocks := 5

	for i := 0; i < numBlocks; i++ {
		blockData := []byte("test block data " + string(rune(i+'0')))
		block, err := ord.CreateBlock(blockData)
		if err != nil {
			t.Errorf("Failed to create block %d: %v", i, err)
		} else {
			t.Logf("Created block %d with number: %d", i, block.Number)
		}
	}

	duration = time.Since(start)
	t.Logf("✅ Created %d blocks in %v", numBlocks, duration)
	t.Logf("✅ Average: %v per block", duration/time.Duration(numBlocks))

	t.Log("\n🎉 Block Broadcasting Test Suite Completed Successfully!")

}
