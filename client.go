package network

import (
	"github.com/galaco/bitbuf"
	"github.com/galaco/gource-network/protocol"
	"github.com/galaco/gource-network/protocol/udp"
	"log"
)

// Networking client
type Client struct {
	conn protocol.IProtocol
	info *ClientInfo
	packetTypeHandlers map[int]func(packet protocol.IPacket)
}

// Connect to remote server
func (client *Client) Connect(host string, port string) error {
	err := client.conn.Connect(host, port)

	if err == nil {
		return err
	}

	return client.Reconnect()
}

// Disconnect from the currently connected server
func (client *Client) Disconnect() {
	client.conn.Disconnect()
}

func (client *Client) Reconnect() error {
	return nil
}

// Listens for data from the server forever
// Creates a goroutine that recieves data from the server continually
// It sends packet data to registered callback functions to allow other
// routines to handle received packets.
// Callbacks should not process the packets themselves; expect crashes or bottlenecks
// if they do.
func (client *Client) Listen() {
	go func() {
		for true {
			pkt,err := client.conn.Receive()
			if err != nil {
				continue
			}
			if ok := client.packetTypeHandlers[pkt.Type()]; ok != nil {
				client.packetTypeHandlers[pkt.Type()](pkt)
			} else {
				client.unhandledPacket(pkt)
			}
		}
	}()
}

// Send some data to the connected server
func (client *Client) SendPacket(packetType int, data []byte) {
	//Assemble packet

}

// Register a callback to make use of the received packet
// NOTE:
// The callback gets executed in the net receiver routine.
// CALLBACKS SHOULD EXIST ONLY TO TRANSFER PACKET DATA OUT OF THIS ROUTINE
func (client *Client) RegisterPacketHandler(packetType int, callback func(packet protocol.IPacket)) {
	client.packetTypeHandlers[packetType] = callback
}

func (client *Client) unhandledPacket(packet protocol.IPacket) {
	log.Printf("Unhandled packet. Type: %d, contents: %s\n", packet.Type(), packet.ToBytes())
}

func (client *Client) QueryStatus() []byte {
	c := make (chan []byte)
	callStep := func(step int, c chan []byte) {
		switch step {
		case 1:
			//Step 1. Challenge request
			buf := bitbuf.NewWriter(100)
			buf.WriteUint8(255)
			buf.WriteUint8(255)
			buf.WriteUint8(255)
			buf.WriteUint8(255)
			buf.WriteByte('T')
			buf.WriteString("Source Engine Query\x00")
			client.conn.Send(udp.NewPacket(-1, buf.Data()[:buf.BytesWritten()]))
		case 2:
			pkt,_ := client.conn.Receive()
			c <- pkt.ToBytes()
		}
	}
	go callStep(2, c)
	go callStep(1, c)

	result := <- c

	return result
}

func NewClient(serverProtocol protocol.IProtocol, info *ClientInfo) *Client {
	return &Client{
		conn: serverProtocol,
		info: info,
		packetTypeHandlers: map[int]func(packet protocol.IPacket){},
	}
}