package udp

import (
	"github.com/galaco/gource-network/protocol"
	"net"
)

type Client struct {
	conn net.Conn
}

// Disconnect from the current server
func (client *Client) Disconnect() {
	client.conn.Close()
}

// Connect to a server.
// Creates a UDP connection then calls the back and forth to identify
// this client
func (client *Client) Connect(host string, port string) error {
	var err error
	client.conn,err = net.Dial("udp", host + ":" + port)
	if err != nil {
		return err
	}

	client.Reconnect()

	return nil
}

// Send a preconstructed packet to the server
func (client *Client) Send(packet protocol.IPacket) error {
	_,err := client.conn.Write(packet.ToBytes())
	return err
}

// Wait for a packet to be received from the server
func (client *Client) Receive() (protocol.IPacket,error) {
	buf := make([]byte, 2048)
	// listen for packet data
	_,err := client.conn.Read(buf)

	if err != nil {
		return nil,err
	}

	// parse it into a packet

	return NewPacket(0, buf), nil
}

// Perform initial connection back and forth
// with server
// See README.md for an explanation on how this works
func (client *Client) Reconnect() error {
	return nil
}

func NewClient() *Client {
	return &Client{}
}