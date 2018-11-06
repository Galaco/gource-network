package main

import (
	"bufio"
	"fmt"
	"github.com/BenLubar/steamworks"
	"github.com/galaco/bitbuf"
	"github.com/galaco/gource-network"
	"github.com/galaco/gource-network/protocol/udp"
	"log"
	"os"
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

	registerHandlers(client)

	// Performs the handshake; and provides information about
	// server status (e.g. current map)
	err = client.Connect("142.44.143.138", "27015")
	if err != nil {
		log.Fatal(err)
	}

	client.Listen()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter anything to disconnect: ")
	text, _ := reader.ReadString('\n')
	fmt.Println(text)

	//for true {
	//	log.Println("listening")
	//	time.Sleep(5 * time.Second)
	//}

	client.Disconnect()

	log.Println("end")
}



func registerHandlers(client *network.Client) {
	client.RegisterPacketHandler(udp.TypeNOP, func(packet *bitbuf.Reader) {
		log.Println("nop")
	})
	client.RegisterPacketHandler(udp.TypeDisconnect, func(packet *bitbuf.Reader) {
		log.Println("disconnected")
	})
}