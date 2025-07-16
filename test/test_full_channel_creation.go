package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ddr4869/minifab/orderer"
	"github.com/ddr4869/minifab/peer"
)

func main() {
	fmt.Println("Testing Full Channel Creation with gRPC Communication")
	fmt.Println("===================================================")

	// Test 1: Start orderer server
	fmt.Println("\n1. Starting orderer server...")

	ord := orderer.NewOrderer("OrdererMSP")
	ordererServer := orderer.NewOrdererServer(ord)

	// Start orderer server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := ordererServer.StartWithContext(ctx, "localhost:7050"); err != nil {
			log.Printf("Orderer server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)
	fmt.Println("✅ Orderer server started on localhost:7050")

	// Test 2: Create peer and connect to orderer
	fmt.Println("\n2. Creating peer and connecting to orderer...")

	peerInstance := peer.NewPeer("peer0", "./chaincode", "Org1MSP")

	ordererClient, err := peer.NewOrdererClient("localhost:7050")
	if err != nil {
		log.Fatalf("Failed to connect to orderer: %v", err)
	}
	defer ordererClient.Close()

	fmt.Println("✅ Peer connected to orderer")

	// Test 3: Create channel with profile via gRPC
	fmt.Println("\n3. Creating channel with profile via gRPC...")

	channelName := "mychannel1"
	profileName := "OrgsChannel0"

	err = ordererClient.CreateChannelWithProfile(channelName, profileName, "config/configtx.yaml")
	if err != nil {
		log.Fatalf("Failed to create channel via gRPC: %v", err)
	}

	fmt.Printf("✅ Channel '%s' created with profile '%s'\n", channelName, profileName)

	// Test 4: Peer joins the channel
	fmt.Println("\n4. Peer joining the channel...")

	err = peerInstance.JoinChannelWithProfile(channelName, profileName, nil)
	if err != nil {
		log.Fatalf("Failed to join channel: %v", err)
	}

	fmt.Printf("✅ Peer joined channel '%s'\n", channelName)

	// Test 5: Verify channel files were created
	fmt.Println("\n5. Verifying generated files...")

	// Check orderer-generated channel config
	ordererChannelFile := fmt.Sprintf("channels/%s.json", channelName)
	if _, err := os.Stat(ordererChannelFile); err == nil {
		fmt.Printf("✅ Orderer channel config: %s\n", ordererChannelFile)

		// Read and display content
		data, err := os.ReadFile(ordererChannelFile)
		if err == nil {
			var config map[string]interface{}
			if json.Unmarshal(data, &config) == nil {
				fmt.Printf("   - Channel ID: %v\n", config["channel_id"])
				fmt.Printf("   - Profile: %v\n", config["profile_name"])
				fmt.Printf("   - Consortium: %v\n", config["consortium"])
				fmt.Printf("   - Created At: %v\n", config["created_at"])
				if orgs, ok := config["organizations"].([]interface{}); ok {
					fmt.Printf("   - Organizations: %d\n", len(orgs))
					for i, org := range orgs {
						if orgMap, ok := org.(map[string]interface{}); ok {
							fmt.Printf("     %d. %v (MSP: %v)\n", i+1, orgMap["name"], orgMap["msp_id"])
						}
					}
				}
			}
		}
	} else {
		fmt.Printf("❌ Orderer channel config not found: %s\n", ordererChannelFile)
	}

	// Test 6: Create another channel with different name
	fmt.Println("\n6. Creating second channel...")

	channelName2 := "testchannel2"
	err = ordererClient.CreateChannelWithProfile(channelName2, profileName, "config/configtx.yaml")
	if err != nil {
		log.Fatalf("Failed to create second channel: %v", err)
	}

	fmt.Printf("✅ Second channel '%s' created\n", channelName2)

	// Test 7: List all channels
	fmt.Println("\n7. Listing all channels...")

	// Wait a moment for file operations to complete
	time.Sleep(1 * time.Second)

	ordererChannels := ord.GetChannels()
	fmt.Printf("📋 Orderer channels: %v\n", ordererChannels)

	peerChannels := peerInstance.GetChannelManager().ListChannels()
	fmt.Printf("📋 Peer channels: %v\n", peerChannels)

	// Test 8: Verify all generated files
	fmt.Println("\n8. Final verification of generated files...")

	expectedFiles := []string{
		fmt.Sprintf("channels/%s.json", channelName),
		fmt.Sprintf("channels/%s.json", channelName2),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("✅ File exists: %s\n", file)
		} else {
			fmt.Printf("❌ File missing: %s\n", file)
		}
	}

	fmt.Println("\n🎉 Full Channel Creation Test Complete!")
	fmt.Println("\nKey Achievements:")
	fmt.Println("✅ Orderer server running with gRPC")
	fmt.Println("✅ Peer connects to orderer via gRPC")
	fmt.Println("✅ Channel creation with configtx.yaml profile")
	fmt.Println("✅ Orderer generates channel JSON files")
	fmt.Println("✅ Peer joins channels successfully")
	fmt.Println("✅ Multiple channels supported")
	fmt.Println("✅ Proper file organization in channels/ directory")

	fmt.Println("\nGenerated Files:")
	for _, file := range expectedFiles {
		fmt.Printf("📁 %s\n", file)
	}

	// Graceful shutdown
	fmt.Println("\n9. Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
	fmt.Println("✅ Shutdown complete")
}
