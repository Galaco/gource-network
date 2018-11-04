package main

import (
	"github.com/BenLubar/steamworks"
	"github.com/galaco/gource-network"
	"github.com/galaco/gource-network/protocol"
	"github.com/galaco/gource-network/protocol/udp"
	"log"
)

func main() {
	// REQUIRES STEAM RUNNING
	err := steamworks.InitClient(true)
	if err != nil {
		log.Println(err)
	}

	client := network.NewClient(udp.NewClient(), &network.ClientInfo{
		ClientChallenge: 0x0B5B1842,
		Name: "gource",
		GameVersion: "2000",
		FakeSteamId: 0x12345678,
	})

	// Register custom handlers for different packet types
	client.RegisterPacketHandler(udp.TypeSvcPrint, func(packet protocol.IPacket) {
		// this can do whatever, including pass back to the default handler
		//log.Println(string(packet.ToBytes()))
	})

	// Performs the handshake; and provides information about
	// server status (e.g. current map)
	err = client.Connect("151.80.230.149", "27015")
	if err != nil {
		log.Fatal(err)
	}

	client.Reconnect()

	client.Disconnect()

	log.Println("end")
}