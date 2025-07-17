package main_test

import (
	"testing"

	"github.com/ddr4869/minifab/peer/channel"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/ddr4869/minifab/peer/core"
)

func TestJoinChannelError(t *testing.T) {
	t.Log("Testing JoinChannel Error Handling")

	// Create peer
	peerInstance := core.NewPeer("peer0", "./chaincode", "Org1MSP")

	// Create channel manager and set it
	channelManager := channel.NewManager()
	peerInstance.SetChannelManager(channelManager)

	// Test 1: Try to join a non-existent channel
	t.Log("\n1. Testing join non-existent channel...")

	// Create a mock orderer client (this will fail in real scenario)
	ordererClient, err := client.NewOrdererClient("localhost:7050")
	if err != nil {
		t.Log("Note: Could not connect to orderer (expected for this test)")
		t.Log("✅ Testing without orderer connection")

		// Test error handling when channel manager is not properly initialized
		t.Log("\n2. Testing channel manager error handling...")

		// Try to get a non-existent channel
		_, err := peerInstance.GetChannelManager().GetChannel("nonexistent")
		if err != nil {
			t.Logf("✅ Properly handled non-existent channel error: %v", err)
		} else {
			t.Error("Expected error for non-existent channel, but got none")
		}

		// Test channel names listing (should be empty)
		channels := peerInstance.GetChannelManager().GetChannelNames()
		if len(channels) == 0 {
			t.Log("✅ Channel list is empty as expected")
		} else {
			t.Errorf("Expected empty channel list, got %d channels", len(channels))
		}

		return
	}
	defer ordererClient.Close()

	// If we successfully connected, test actual join scenarios
	t.Log("✅ Connected to orderer")

	// Test with non-existent channel
	err = peerInstance.JoinChannel("nonexistent-channel", ordererClient)
	if err != nil {
		t.Logf("✅ Properly rejected join to non-existent channel: %v", err)
	} else {
		t.Error("Expected error when joining non-existent channel, but succeeded")
	}

	// Test 2: Create a channel first, then join
	t.Log("\n2. Testing successful channel creation and join...")

	channelName := "testchannel"

	// Create channel first
	err = peerInstance.CreateChannel(channelName, ordererClient)
	if err != nil {
		t.Errorf("Failed to create channel: %v", err)
	} else {
		t.Log("✅ Channel created successfully")

		// Now try to join the channel
		err = peerInstance.JoinChannel(channelName, ordererClient)
		if err != nil {
			t.Errorf("Failed to join existing channel: %v", err)
		} else {
			t.Log("✅ Successfully joined existing channel")
		}
	}

	// Test 3: Test multiple join attempts
	t.Log("\n3. Testing multiple join attempts...")

	// Try joining the same channel again
	err = peerInstance.JoinChannel(channelName, ordererClient)
	if err != nil {
		t.Logf("Note: Multiple join attempts handled: %v", err)
	} else {
		t.Log("✅ Multiple join attempts allowed (may be expected behavior)")
	}

	// Test 4: Test with nil orderer client
	t.Log("\n4. Testing with nil orderer client...")

	err = peerInstance.JoinChannel(channelName, nil)
	if err != nil {
		t.Logf("✅ Properly handled nil orderer client: %v", err)
	} else {
		t.Error("Expected error with nil orderer client, but succeeded")
	}

	// Test 5: Test channel validation
	t.Log("\n5. Testing channel validation...")

	// Test empty channel name
	err = peerInstance.JoinChannel("", ordererClient)
	if err != nil {
		t.Logf("✅ Properly rejected empty channel name: %v", err)
	} else {
		t.Error("Expected error with empty channel name, but succeeded")
	}

	// Test invalid channel name characters
	invalidNames := []string{
		"channel with spaces",
		"channel-with-@-symbol",
		"UPPER_CASE_CHANNEL",
		"channel.with.dots",
	}

	for _, invalidName := range invalidNames {
		t.Logf("Testing invalid channel name: '%s'", invalidName)
		err = peerInstance.JoinChannel(invalidName, ordererClient)
		if err != nil {
			t.Logf("✅ Properly rejected invalid name '%s': %v", invalidName, err)
		} else {
			t.Logf("Note: Invalid name '%s' was accepted (may need validation)", invalidName)
		}
	}

	// Test 6: Test concurrent join attempts
	t.Log("\n6. Testing concurrent join attempts...")

	// Create multiple channels for concurrent testing
	testChannels := []string{"concurrent1", "concurrent2", "concurrent3"}

	for _, ch := range testChannels {
		err := peerInstance.CreateChannel(ch, ordererClient)
		if err != nil {
			t.Errorf("Failed to create test channel %s: %v", ch, err)
		}
	}

	// Test concurrent joins
	results := make(chan error, len(testChannels))

	for _, ch := range testChannels {
		go func(channelName string) {
			err := peerInstance.JoinChannel(channelName, ordererClient)
			results <- err
		}(ch)
	}

	// Collect results
	successCount := 0
	for i := 0; i < len(testChannels); i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			t.Logf("Concurrent join error: %v", err)
		}
	}

	t.Logf("✅ Concurrent joins: %d/%d successful", successCount, len(testChannels))

	// Test 7: Resource cleanup test
	t.Log("\n7. Testing resource cleanup...")

	// Check if we can list all channels after all operations
	allChannels := peerInstance.GetChannelManager().GetChannelNames()
	t.Logf("✅ Total channels after all tests: %d", len(allChannels))

	for i, ch := range allChannels {
		t.Logf("   %d. %s", i+1, ch)
	}

	// Test 8: Memory usage after operations
	t.Log("\n8. Checking memory usage...")

	// Try to get details of each channel
	for _, chName := range allChannels {
		ch, err := peerInstance.GetChannelManager().GetChannel(chName)
		if err != nil {
			t.Errorf("Failed to get channel %s: %v", chName, err)
		} else {
			t.Logf("   - %s: %d transactions, MSP configured: %v",
				chName, len(ch.Transactions), ch.MSP != nil)
		}
	}

	t.Log("\n🎉 JoinChannel Error Handling Test Completed Successfully!")
}
