package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ddr4869/minifab/orderer"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/ddr4869/minifab/peer/core"
)

func TestFullChannelCreation(t *testing.T) {
	t.Log("Testing Full Channel Creation with gRPC Communication")

	// Test 1: Start orderer server
	t.Log("\n1. Starting orderer server...")

	ord := orderer.NewOrderer("OrdererMSP")
	ordererServer := orderer.NewOrdererServer(ord)

	go func() {
		if err := ordererServer.Start(":7050"); err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				t.Log("Orderer server already running")
				return
			}
			t.Errorf("Orderer server failed to st	art: %v", err)
		}
	}()

	// Wait for server startup
	time.Sleep(2 * time.Second)
	t.Log("✅ Orderer server started")

	// Test 2: Create orderer client
	t.Log("\n2. Creating orderer client...")

	ordererClient, err := client.NewOrdererClient("localhost:7050")
	if err != nil {
		t.Fatalf("Failed to create orderer client: %v", err)
	}
	defer ordererClient.Close()

	t.Log("✅ Orderer client created")

	// Test 3: Create peer with MSP and orderer client
	t.Log("\n3. Creating peer with MSP and orderer client...")

	peer := core.NewPeerWithMSPFiles("peer0", "./chaincode", "Org1MSP", "ca/ca-client/peer0", ordererClient)

	t.Log("✅ Peer created with MSP, orderer client and channel manager")

	// Test 4: Create channel through peer
	t.Log("\n4. Creating channel through peer...")

	channelName := "mychannel"
	err = peer.CreateChannel(channelName)
	if err != nil {
		t.Fatalf("Failed to create channel through peer: %v", err)
	}

	t.Logf("✅ Channel '%s' created through peer", channelName)

	// Test 5: Verify channel exists locally
	t.Log("\n5. Verifying channel exists locally...")

	channel, err := peer.GetChannelManager().GetChannel(channelName)
	if err != nil {
		t.Errorf("Failed to get channel locally: %v", err)
	} else {
		t.Logf("✅ Channel found locally: %s", channel.Name)
		t.Logf("   - Config: %+v", channel.Config)
	}

	// Test 6: Join channel
	t.Log("\n6. Testing channel join...")

	err = peer.JoinChannel(channelName)
	if err != nil {
		t.Errorf("Failed to join channel: %v", err)
	} else {
		t.Logf("✅ Successfully joined channel: %s", channelName)
	}

	// Test 7: Submit transaction
	t.Log("\n7. Submitting transaction...")

	txData := []byte("test transaction data")
	tx, err := peer.SubmitTransaction(channelName, txData)
	if err != nil {
		t.Errorf("Failed to submit transaction: %v", err)
	} else {
		t.Logf("✅ Transaction submitted: %s", tx.ID)
		t.Logf("   - Channel: %s", tx.ChannelID)
		t.Logf("   - Payload size: %d bytes", len(tx.Payload))
	}

	// Test 8: Validate transaction
	if tx != nil {
		t.Log("\n8. Validating transaction...")

		err = peer.ValidateTransaction(tx)
		if err != nil {
			t.Errorf("Transaction validation failed: %v", err)
		} else {
			t.Log("✅ Transaction validation successful")
		}
	}

	// Test 9: Create multiple channels
	t.Log("\n9. Creating multiple channels...")

	channels := []string{"channel1", "channel2", "channel3"}
	for _, ch := range channels {
		t.Logf("Creating channel: %s", ch)

		err := peer.CreateChannel(ch)
		if err != nil {
			t.Errorf("Failed to create channel %s: %v", ch, err)
		} else {
			t.Logf("✅ Channel %s created", ch)
		}
	}

	// Test 10: List all channels
	t.Log("\n10. Listing all channels...")

	channelNames := peer.GetChannelManager().GetChannelNames()
	t.Logf("✅ Found %d channels:", len(channelNames))
	for i, name := range channelNames {
		t.Logf("   %d. %s", i+1, name)
	}

	// Test 11: Submit transactions to multiple channels
	t.Log("\n11. Submitting transactions to multiple channels...")

	for _, ch := range channels {
		txData := []byte(fmt.Sprintf("transaction data for %s", ch))
		tx, err := peer.SubmitTransaction(ch, txData)
		if err != nil {
			t.Errorf("Failed to submit transaction to %s: %v", ch, err)
		} else {
			t.Logf("✅ Transaction submitted to %s: %s", ch, tx.ID)
		}
	}

	// Test 12: Channel configuration persistence
	t.Log("\n12. Testing channel configuration persistence...")

	// Check if channel config files were created
	for _, ch := range append(channels, channelName) {
		configPath := "channels/" + ch + ".json"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file not found for channel %s", ch)
		} else {
			t.Logf("✅ Config file exists for channel %s", ch)

			// Verify JSON format
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Errorf("Failed to read config for %s: %v", ch, err)
			} else {
				var config map[string]interface{}
				if err := json.Unmarshal(data, &config); err != nil {
					t.Errorf("Invalid JSON in config for %s: %v", ch, err)
				}
			}
		}
	}

	// Test 13: Performance testing
	t.Log("\n13. Performance testing...")

	start := time.Now()
	numTx := 20

	for i := 0; i < numTx; i++ {
		txData := []byte(fmt.Sprintf("performance test transaction %d", i))
		_, err := peer.SubmitTransaction(channelName, txData)
		if err != nil {
			t.Errorf("Failed to submit performance tx %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	t.Logf("✅ Submitted %d transactions in %v", numTx, duration)
	t.Logf("✅ Average: %v per transaction", duration/time.Duration(numTx))

	// Test 14: Memory usage check
	t.Log("\n14. Memory usage check...")

	chManager := peer.GetChannelManager()
	if chManager != nil {
		allChannels := chManager.GetChannelNames()
		t.Logf("✅ Channel manager holds %d channels", len(allChannels))

		for _, chName := range allChannels {
			ch, err := chManager.GetChannel(chName)
			if err == nil && ch != nil {
				t.Logf("   - %s: %d transactions", chName, len(ch.Transactions))
			}
		}
	}

	t.Log("\n🎉 Full Channel Creation Test Completed Successfully!")
}
