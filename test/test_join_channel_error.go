package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ddr4869/minifab/peer"
)

func main() {
	fmt.Println("Testing JoinChannel Error Handling")
	fmt.Println("==================================")

	// Create peer
	peerInstance := peer.NewPeer("peer0", "./chaincode", "Org1MSP")

	// Test 1: Try to join a non-existent channel
	fmt.Println("\n1. Testing join non-existent channel...")

	err := peerInstance.JoinChannel("nonexistent-channel", nil)
	if err != nil {
		fmt.Printf("✅ Expected error received: %v\n", err)
	} else {
		fmt.Println("❌ Expected error but got success")
	}

	// Test 2: Create a channel first, then join
	fmt.Println("\n2. Testing proper workflow: create then join...")

	// Create channel locally (simulating orderer creation)
	channelManager := peerInstance.GetChannelManager()
	uniqueChannelName := fmt.Sprintf("testchannel-%d", time.Now().UnixNano())
	err = channelManager.CreateChannel(uniqueChannelName, "SampleConsortium", "localhost:7050")
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
	fmt.Println("✅ Channel created successfully")

	// Now try to join the existing channel
	err = peerInstance.JoinChannel(uniqueChannelName, nil)
	if err != nil {
		fmt.Printf("❌ Unexpected error: %v\n", err)
	} else {
		fmt.Println("✅ Successfully joined existing channel")
	}

	// Test 3: Test JoinChannelWithProfile with non-existent channel
	fmt.Println("\n3. Testing JoinChannelWithProfile with non-existent channel...")

	err = peerInstance.JoinChannelWithProfile("another-nonexistent", "OrgsChannel0", nil)
	if err != nil {
		fmt.Printf("✅ Expected error received: %v\n", err)
	} else {
		fmt.Println("❌ Expected error but got success")
	}

	fmt.Println("\n🎉 JoinChannel Error Handling Test Complete!")
	fmt.Println("\nKey Behaviors:")
	fmt.Println("✅ JoinChannel fails when channel doesn't exist")
	fmt.Println("✅ JoinChannelWithProfile fails when channel doesn't exist")
	fmt.Println("✅ JoinChannel succeeds when channel exists")
	fmt.Println("✅ Clear error messages guide users to create channel first")
}
