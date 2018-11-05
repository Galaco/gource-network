package network

import (
	"bytes"
	"encoding/binary"
	"github.com/galaco/bitbuf"
	"github.com/galaco/gource-network/internal"
	"github.com/galaco/gource-network/protocol"
	"github.com/galaco/gource-network/protocol/udp"
	"log"
)

// Networking client
type Client struct {
	conn protocol.IProtocol
	channel *Channel
	info *ClientInfo
	packetTypeHandlers map[uint8]func(*bitbuf.Reader)

	connectionStep int
}

// Connect to remote server
func (client *Client) Connect(host string, port string) error {
	err := client.conn.Connect(host, port)

	return err
}

// Disconnect from the currently connected server
func (client *Client) Disconnect() {
	client.conn.Disconnect()
}

func (client *Client) Reconnect() error {
	var err error
	client.connectionStep,err = client.conn.Reconnect()
	return err
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
			buf := bitbuf.NewReader(pkt.ToBytes())

			flags := client.channel.ReadPacketHeader(buf)
			if flags == -1 {
				log.Println("Bad packet!")
				return
			}
			if flags & packetFlagReliable != 0 {
				bits,_ := buf.ReadBits(3)
				var bit uint32
				binary.Read(bytes.NewBuffer(bits), binary.LittleEndian, &bit)
				bit = 1 << bit

				for i := 0; i < maxStreams; i++ {
					bits,_ := buf.ReadBits(1)
					if bits[0] != 0 {
						//if (!netchan->ReadSubChannelData(recvdata, i))
						//{
						//	return 0;
						//}
					}
				}
				if uint32(client.channel.inReliableState) & bit != 0 {
					client.channel.inReliableState = int32(uint32(client.channel.inReliableState) & ^bit)
				} else {
					client.channel.inReliableState = int32(uint32(client.channel.inReliableState) | bit)
				}

				//for i := 0; i<maxStreams; i++ {
				//	if (!netchan->CheckReceivingList(i)) {
				//		continue
				//	}
				//}
			}

			if buf.BitsRead() < buf.Size() {
				bt,_ := buf.ReadBits(6)
				packetType := uint8(bt[0])

				if ok := client.packetTypeHandlers[packetType]; ok != nil {
					client.packetTypeHandlers[packetType](buf)
				} else {
					log.Printf("Unhandled packet. Type: %d\n", packetType)
					client.unhandledPacket(buf)
				}
			}

			//static bool neededfragments = false;
			//
			//if (netchan->NeedsFragments() || flags&PACKET_FLAG_TABLES)
			//{
			//	neededfragments = true;
			//	NET_RequestFragments();
			//}

			continue
		}
	}()
}

// Send some data to the connected server
func (client *Client) SendPacket(data *bitbuf.Writer) {
	// @TODO add subchannel support

	subchans := 0

	flags := uint8(0)


	//Assemble packet
	datagram := bitbuf.NewWriter(2048)
	datagram.WriteInt32(client.channel.outSequenceNrAck)
	datagram.WriteInt32(client.channel.inSequenceNr)

	flagPos := datagram.BitsWritten()
	datagram.WriteByte(0)
	checksumPos := datagram.BitsWritten()
	datagram.WriteUint16(0)

	checkSumStart := datagram.BytesWritten()
	datagram.WriteByte(byte(client.channel.inReliableState))


	if subchans > 0 {
		flags |= packetFlagReliable
	}

	datagram.WriteBytes(data.Data()[:data.BytesWritten()])

	// constant
	minRoutablePayload := 16
	for datagram.BytesWritten() < minRoutablePayload && datagram.BitsWritten() % 8 != 0 {
		datagram.WriteUnsignedBitInt32(0, 6)
	}

	curPos := datagram.BitsWritten()
	datagram.Seek(flagPos)
	datagram.WriteByte(flags)
	datagram.Seek(curPos)

	ccData := datagram.Data()[checkSumStart:datagram.BytesWritten()]

	if len(ccData) > 0 {
		checkSum := internal.CRC32(ccData)
		datagram.Seek(checksumPos)
		datagram.WriteUint16(checkSum)
		datagram.Seek(curPos)

		client.channel.outSequenceNr++

		client.conn.Send(udp.NewPacket(datagram.Data()[:datagram.BytesWritten()]))
	}
}

// Register a callback to make use of the received packet
// NOTE:
// The callback gets executed in the net receiver routine.
// CALLBACKS SHOULD EXIST ONLY TO TRANSFER PACKET DATA OUT OF THIS ROUTINE
func (client *Client) RegisterPacketHandler(packetType uint8, callback func(packet *bitbuf.Reader)) {
	client.packetTypeHandlers[packetType] = callback
}

func (client *Client) unhandledPacket(packet *bitbuf.Reader) {
	log.Printf("Unhandled packet. contents: %s\n", packet.Data())
}

func (client *Client) QueryStatus() []byte {
	c := make (chan []byte)
	callStep := func(step int, c chan []byte) {
		switch step {
		case 1:
			//Step 1. Challenge request
			buf := bitbuf.NewWriter(100)
			buf.WriteByte(255)
			buf.WriteByte(255)
			buf.WriteByte(255)
			buf.WriteByte(255)
			buf.WriteByte('T')
			buf.WriteString("Source Engine Query\x00")
			client.conn.Send(udp.NewPacket(buf.Data()[:buf.BytesWritten()]))
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

func (client *Client) registerInternalHandlers() {
	unknownCounter := int(20)
	client.RegisterPacketHandler(udp.TypeNetSignonState, func(packet *bitbuf.Reader) {
		state,_ := packet.ReadUint8()
		serverCount,_ := packet.ReadInt32()

		if client.channel.signOnState == int32(state) {
			return
		}
		client.channel.serverCount = serverCount
		client.channel.signOnState = int32(state)

		if state == 3 {
			senddata := bitbuf.NewWriter(1000)
			senddata.WriteUnsignedBitInt32(8, 6)
			senddata.WriteInt32(serverCount)
			senddata.WriteInt32(-2030366758)
			senddata.WriteUnsignedBitInt32(1, 1)
			senddata.WriteInt32(1337)

			unknownCounter++

			senddata.WriteUnsignedBitInt32(0, 20)
			client.SendPacket(senddata)

			senddata = bitbuf.NewWriter(1000)
			senddata.WriteUnsignedBitInt32(0, 6)
			senddata.WriteUnsignedBitInt32(6, 6)
			senddata.WriteByte(state)
			senddata.WriteInt32(serverCount)
			client.SendPacket(senddata)
		}

		if client.connectionStep != 0 {
			senddata := bitbuf.NewWriter(1000)
			if (state == 4) && false {
				senddata.WriteUnsignedBitInt32(12, 6)
				for i := 0; i < 32; i++ {
					senddata.WriteUnsignedBitInt32(1, 32)
				}
			}

			senddata.WriteUnsignedBitInt32(6, 6)
			senddata.WriteByte(state)
			senddata.WriteInt32(serverCount)

			client.SendPacket(senddata)

			return
		}

		senddata := bitbuf.NewWriter(1000)

		senddata.WriteUnsignedBitInt32(6, 6)
		senddata.WriteByte(state)
		senddata.WriteInt32(serverCount)
	})
}

func NewClient(serverProtocol protocol.IProtocol, info *ClientInfo) *Client {
	c := &Client{
		channel: NewChannel(),
		conn: serverProtocol,
		info: info,
		packetTypeHandlers: map[uint8]func(packet *bitbuf.Reader){},
		connectionStep: 0,
	}
	c.registerInternalHandlers()

	return c
}