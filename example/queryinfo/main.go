package main

import (
	"github.com/galaco/gource-network"
	"github.com/galaco/gource-network/protocol"
	"github.com/galaco/gource-network/protocol/udp"
	"log"
	"strings"
)

func main() {
	client := network.NewClient(udp.NewClient(), &network.ClientInfo{
		ClientChallenge: 0x0B5B1842,
		Name: "gource",
		GameVersion: "2000",
		FakeSteamId: 0x12345678,
	})

	// Register custom handlers for different packet types
	client.RegisterPacketHandler(udp.TypeSvcPrint, func(packet protocol.IPacket) {
		// this can do whatever, including pass back to the default handler
		log.Println(string(packet.ToBytes()))
	})

	// Performs the handshake; and provides information about
	// server status (e.g. current map)
	err := client.Connect("151.80.230.149", "27015")
	if err != nil {
		log.Fatal(err)
	}

	props := strings.Split(string(client.QueryStatus()[6:]), "\x00")
	log.Println("Server name: " + props[0])
	log.Println("Map: " + props[1])
	log.Println("Game id: " + props[2])
	log.Println("Game mode: " + props[3])
	log.Printf("Players: %d/%d\n", uint8(props[5][0]), uint8(props[5][1]))

	client.Disconnect()

	log.Println("end")
}