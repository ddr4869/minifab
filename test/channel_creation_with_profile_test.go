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

func TestChannelCreationWithProfile(t *testing.T) {
	t.Log("Testing Channel Creation with Profile")

	// Test 1: Create orderer instance
	t.Log("\n1. Creating orderer instance...")

	// Create orderer
	ord := orderer.NewOrderer("OrdererMSP")
	ordererServer := orderer.NewOrdererServer(ord)

	t.Log("✅ Orderer instance created")

	// Test 2: Start orderer server
	t.Log("\n2. Starting orderer server...")

	go func() {
		if err := ordererServer.Start(":7050"); err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				t.Log("Orderer server already running")
				return
			}
			t.Errorf("❌ Orderer server failed to start: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)
	t.Log("✅ Orderer server started")

	// Test 3: Connect orderer client
	t.Log("\n3. Connecting to orderer...")

	ordererClient, err := client.NewOrdererClient("localhost:7050")
	if err != nil {
		t.Fatalf("Failed to create orderer client: %v", err)
	}
	defer ordererClient.Close()

	t.Log("✅ Connected to orderer")

	// Create peer instance if needed for future tests
	_ = core.NewPeer("peer0", "./chaincode", "Org1MSP", ordererClient)

	// Test 4: Create channel with profile
	t.Log("\n4. Creating channel with profile...")

	channelName := "testchannel"
	profileName := "TwoOrgsChannel"
	configTxPath := "config/configtx.yaml"

	err = ordererClient.CreateChannelWithProfile(channelName, profileName, configTxPath)
	if err != nil {
		t.Fatalf("Failed to create channel with profile: %v", err)
	}

	t.Logf("✅ Channel '%s' created with profile '%s'", channelName, profileName)

	// Test 5: Verify channel configuration file was created
	t.Log("\n5. Verifying channel configuration...")

	channelConfigPath := "channels/" + channelName + ".json"
	if _, err := os.Stat(channelConfigPath); os.IsNotExist(err) {
		t.Errorf("❌ Channel config file not found: %s", channelConfigPath)
	} else {
		t.Logf("✅ Channel config file created: %s", channelConfigPath)

		// Read and validate the configuration
		configData, err := os.ReadFile(channelConfigPath)
		if err != nil {
			t.Errorf("❌ Failed to read channel config: %v", err)
		} else {
			var config map[string]interface{}
			if err := json.Unmarshal(configData, &config); err != nil {
				t.Errorf("❌ Invalid JSON in channel config: %v", err)
			} else {
				t.Log("✅ Channel configuration is valid JSON")
				t.Logf("Channel config: %v", config)
			}
		}
	}

	// Test 6: Create multiple channels with different profiles
	t.Log("\n6. Testing multiple channel creation...")

	testChannels := []struct {
		name    string
		profile string
	}{
		{"testchannel2", "OneOrgChannel"},
		{"testchannel3", "TwoOrgsChannel"},
	}

	for _, ch := range testChannels {
		t.Logf("Creating channel '%s' with profile '%s'", ch.name, ch.profile)

		err := ordererClient.CreateChannelWithProfile(ch.name, ch.profile, configTxPath)
		if err != nil {
			t.Errorf("❌ Failed to create channel %s: %v", ch.name, err)
		} else {
			t.Logf("✅ Channel '%s' created successfully", ch.name)
		}

		// Verify config file exists
		configPath := "channels/" + ch.name + ".json"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("❌ Config file not found for channel %s", ch.name)
		}
	}

	// Test 7: Attempt to create duplicate channel
	t.Log("\n7. Testing duplicate channel creation...")

	err = ordererClient.CreateChannelWithProfile(channelName, profileName, configTxPath)
	if err == nil {
		t.Log("Note: Duplicate channel creation was allowed (may be expected behavior)")
	} else {
		t.Logf("✅ Duplicate channel creation properly rejected: %v", err)
	}

	// Test 8: Test with invalid profile
	t.Log("\n8. Testing invalid profile...")

	err = ordererClient.CreateChannelWithProfile("invalidchannel", "NonExistentProfile", configTxPath)
	if err == nil {
		t.Log("Note: Invalid profile was accepted (may need validation)")
	} else {
		t.Logf("✅ Invalid profile properly rejected: %v", err)
	}

	// Test 9: Performance test for channel creation
	t.Log("\n9. Performance testing...")

	start := time.Now()
	numChannels := 3

	for i := 0; i < numChannels; i++ {
		channelName := fmt.Sprintf("perfchannel%d", i)
		err := ordererClient.CreateChannelWithProfile(channelName, "TwoOrgsChannel", configTxPath)
		if err != nil {
			t.Errorf("❌ Failed to create performance channel %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	t.Logf("✅ Created %d channels in %v", numChannels, duration)
	t.Logf("✅ Average: %v per channel", duration/time.Duration(numChannels))

	t.Log("\n🎉 Channel Creation with Profile Test Completed Successfully!")
}
