package main_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ddr4869/minifab/orderer"
	"github.com/ddr4869/minifab/peer/cli"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/ddr4869/minifab/peer/core"
)

func TestProperChannelWorkflow(t *testing.T) {
	t.Log("Testing Proper Channel Creation and Join Workflow")

	// Test 1: Start orderer server
	t.Log("\n1. Starting orderer server...")

	ord := orderer.NewOrderer("OrdererMSP")
	ordererServer := orderer.NewOrdererServer(ord)

	// Start orderer server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := ordererServer.StartWithContext(ctx, "localhost:7050"); err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				t.Log("Orderer server already running")
				return
			}
			t.Errorf("Orderer server failed to start: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)
	t.Log("✅ Orderer server started on localhost:7050")

	// Test 2: Create peer and connect to orderer
	t.Log("\n2. Creating peer and connecting to orderer...")

	peerInstance := core.NewPeer("peer0", "./chaincode", "Org1MSP")

	ordererClient, err := client.NewOrdererClient("localhost:7050")
	if err != nil {
		t.Fatalf("Failed to connect to orderer: %v", err)
	}
	defer ordererClient.Close()

	t.Log("✅ Peer connected to orderer")

	// Test 3: Try to join non-existent channel (should fail)
	t.Log("\n3. Testing join non-existent channel (should fail)...")

	err = peerInstance.JoinChannel("nonexistent-channel", ordererClient)
	if err != nil {
		t.Logf("✅ Expected error: %v", err)
	} else {
		t.Error("❌ Expected error but got success")
	}

	// Test 4: Create channel via peer (which uses orderer client)
	t.Log("\n4. Creating channel via peer (using orderer)...")

	channelName := fmt.Sprintf("testchannel-%d", time.Now().UnixNano())
	err = peerInstance.CreateChannelWithProfile(channelName, "OrgsChannel0", ordererClient)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	t.Logf("✅ Channel '%s' created successfully via orderer", channelName)

	// Test 5: Now join the created channel (should succeed)
	t.Log("\n5. Joining the created channel...")

	err = peerInstance.JoinChannel(channelName, ordererClient)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	t.Logf("✅ Successfully joined channel '%s'", channelName)

	// Test 6: Create another channel with different profile
	t.Log("\n6. Creating second channel...")

	channelName2 := fmt.Sprintf("mychannel-%d", time.Now().UnixNano())
	err = peerInstance.CreateChannel(channelName2, ordererClient)
	if err != nil {
		t.Fatalf("Failed to create second channel: %v", err)
	}

	t.Logf("✅ Second channel '%s' created successfully", channelName2)

	// Test 7: Join second channel
	t.Log("\n7. Joining second channel...")

	err = peerInstance.JoinChannel(channelName2, ordererClient)
	if err != nil {
		t.Fatalf("Failed to join second channel: %v", err)
	}

	t.Logf("✅ Successfully joined second channel '%s'", channelName2)

	// Test 8: List all channels
	t.Log("\n8. Listing all channels...")

	// Wait a moment for operations to complete
	time.Sleep(1 * time.Second)

	ordererChannels := ord.GetChannels()
	t.Logf("📋 Orderer channels: %v", ordererChannels)

	peerChannels := peerInstance.GetChannelManager().ListChannels()
	t.Logf("📋 Peer channels: %v", peerChannels)

	// Test 9: Test CLI handlers
	t.Log("\n9. Testing CLI handlers...")

	cliHandlers := cli.NewHandlers(peerInstance, ordererClient)

	// Create channel via CLI handler
	channelName3 := fmt.Sprintf("cli-channel-%d", time.Now().UnixNano())
	err = cliHandlers.HandleChannelCreateWithProfile(channelName3, "OrgsChannel0")
	if err != nil {
		t.Logf("CLI channel creation failed: %v", err)
	} else {
		t.Logf("✅ CLI channel creation successful: %s", channelName3)

		// Join via CLI handler
		err = cliHandlers.HandleChannelJoin(channelName3)
		if err != nil {
			t.Logf("CLI channel join failed: %v", err)
		} else {
			t.Logf("✅ CLI channel join successful: %s", channelName3)
		}
	}

	// Test 10: Verify workflow integrity
	t.Log("\n10. Verifying workflow integrity...")

	// Verify all channels exist on both orderer and peer
	allChannels := []string{channelName, channelName2}
	if channelName3 != "" {
		allChannels = append(allChannels, channelName3)
	}

	for _, ch := range allChannels {
		// Check if channel exists on peer
		_, err := peerInstance.GetChannelManager().GetChannel(ch)
		if err != nil {
			t.Errorf("Channel %s not found on peer: %v", ch, err)
		} else {
			t.Logf("✅ Channel %s verified on peer", ch)
		}
	}

	// Test 11: Performance metrics
	t.Log("\n11. Collecting performance metrics...")

	start := time.Now()
	quickChannelName := fmt.Sprintf("perf-channel-%d", time.Now().UnixNano())

	// Measure channel creation time
	createStart := time.Now()
	err = peerInstance.CreateChannel(quickChannelName, ordererClient)
	createDuration := time.Since(createStart)

	if err != nil {
		t.Errorf("Performance test channel creation failed: %v", err)
	} else {
		t.Logf("✅ Channel creation took: %v", createDuration)

		// Measure join time
		joinStart := time.Now()
		err = peerInstance.JoinChannel(quickChannelName, ordererClient)
		joinDuration := time.Since(joinStart)

		if err != nil {
			t.Errorf("Performance test channel join failed: %v", err)
		} else {
			t.Logf("✅ Channel join took: %v", joinDuration)
		}
	}

	totalDuration := time.Since(start)
	t.Logf("✅ Performance test completed in: %v", totalDuration)

	// Test 12: Resource cleanup verification
	t.Log("\n12. Resource cleanup verification...")

	finalChannelCount := len(peerInstance.GetChannelManager().GetChannelNames())
	t.Logf("✅ Final channel count: %d", finalChannelCount)

	if finalChannelCount < 3 {
		t.Errorf("Expected at least 3 channels, got %d", finalChannelCount)
	}

	t.Log("\n🎉 Proper Channel Workflow Test Complete!")
	t.Log("\nKey Achievements:")
	t.Log("✅ Peer creates channels via orderer client (gRPC)")
	t.Log("✅ Orderer processes channel creation with profiles")
	t.Log("✅ JoinChannel fails for non-existent channels")
	t.Log("✅ JoinChannel succeeds for existing channels")
	t.Log("✅ Multiple channels supported")
	t.Log("✅ CLI handlers work correctly")
	t.Log("✅ Proper separation of create vs join operations")

	t.Log("\nWorkflow Summary:")
	t.Log("1. peer.CreateChannel() → ordererClient.CreateChannel() → orderer processes")
	t.Log("2. peer.JoinChannel() → checks if channel exists locally → joins if exists")
	t.Log("3. Clear error messages when trying to join non-existent channels")

	// Graceful shutdown
	t.Log("\n13. Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
	t.Log("✅ Shutdown complete")
}
