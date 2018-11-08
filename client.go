package network

import (
	"bytes"
	"encoding/binary"
	"github.com/BenLubar/steamworks"
	"github.com/BenLubar/steamworks/steamauth"
	"github.com/galaco/bitbuf"
	"github.com/galaco/gource-network/internal"
	"github.com/galaco/gource-network/protocol"
	"github.com/galaco/gource-network/protocol/udp"
	"log"
	"time"
)

// Networking client
type Client struct {
	conn protocol.IProtocol
	channel *Channel
	info *ClientInfo
	packetTypeHandlers map[uint8]func(*bitbuf.Reader)

	connectionStep int
	signonState int
}

// Connect to remote server
func (client *Client) Connect(host string, port string) error {
	err := client.conn.Connect(host, port)

	return err
}

// Disconnect from the currently connected server
func (client *Client) Disconnect() {
	buf := bitbuf.NewWriter(1024)
	buf.WriteUnsignedBitInt32(1, 8)
	buf.WriteString("Disconnect by User.")
	buf.WriteByte(0)
	client.SendPacket(buf, true)

	client.conn.Disconnect()
}

func (client *Client) Reconnect() error {
	var err error
	client.connectionStep = 1
	return err
}

// Listens for data from the server forever
// Creates a goroutine that recieves data from the server continually
// It sends packet data to registered callback functions to allow other
// routines to handle received packets.
// Callbacks should not process the packets themselves; expect crashes or bottlenecks
// if they do.
func (client *Client) Listen() {
	client.connectionStep = 1

	go func() {
		for true {
			pkt,err := client.conn.Receive()
			if err != nil {
				continue
			}
			buf := bitbuf.NewReader(pkt.ToBytes())

			header,_ := buf.ReadInt32()
			connectionLess := int32(0)

			buf.Reset()

			if header == udp.PacketHeaderFlagQuery {
				connectionLess = 1
			}

			if connectionLess > 0 {
				client.handleConnectionlessPacket(buf, connectionLess)
				continue
			}

			flags := client.channel.ReadPacketHeader(buf)
			if flags == -1 {
				log.Println("Bad packet!")
				continue
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
				bt,_ := buf.ReadByte()
				packetType := uint8(bt)

				log.Printf("Packettype: %d\n", packetType)

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
func (client *Client) SendPacket(data *bitbuf.Writer, asDatagram bool) {
	if asDatagram == true {
		data = client.writePacketHeader(data)
	}
	client.conn.Send(udp.NewPacket(data.Data()[:data.BytesWritten()]))
}

func (client *Client) writePacketHeader(data *bitbuf.Writer) *bitbuf.Writer {
	// @TODO add subchannel support

	subchans := 0

	flags := uint8(0)


	//Assemble packet
	datagram := bitbuf.NewWriter(2048)
	datagram.WriteInt32(client.channel.outSequenceNr)
	datagram.WriteInt32(client.channel.inSequenceNr)

	flagPos := datagram.BitsWritten()
	datagram.WriteUint8(0) //flags
	checksumPos := datagram.BitsWritten()
	datagram.WriteUint16(0) //checksum (crc16)

	checkSumStart := datagram.BytesWritten()

	datagram.WriteByte(byte(client.channel.inReliableState))

	//datagram.WriteUint32(908164834)


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
	}
	// align to byte boundary
	for datagram.BitsWritten() % 8 != 0 {
		datagram.Seek(datagram.BitsWritten() + 1)
	}

	return datagram
}

func (client *Client) handleConnectionlessPacket(packet *bitbuf.Reader, state int32) {
	packet.ReadInt32()

	header := byte(0)
	//id := int32(0)
	//total := uint8(0)
	//number := uint8(0)
	//splitsize := uint8(0)

	if state == 1 {
		header,_ = packet.ReadUint8()
	} else {
		packet.ReadInt32()
		packet.ReadByte()
		packet.ReadByte()
		packet.ReadByte()
	}

	switch header {
	case '9':
		packet.ReadInt32()
		msg,_ := packet.ReadString(1024)
		log.Println("Kicked: " + msg)
		return
	case 'A':
		client.connectionStep = 2
		packet.ReadInt32()
		serverchallenge,_ := packet.ReadInt32()
		ourchallenge,_ := packet.ReadInt32()

		// CREATE NEW PACKET
		buf := bitbuf.NewWriter(1000)
		buf.WriteByte(255)
		buf.WriteByte(255)
		buf.WriteByte(255)
		buf.WriteByte(255)
		buf.WriteByte('k')
		buf.WriteInt32(0x18)
		buf.WriteInt32(0x03)
		buf.WriteInt32(serverchallenge)
		buf.WriteInt32(ourchallenge)
		//buf.WriteUint32(2729496039)
		buf.WriteString("DormantLemon^___") //player name
		buf.WriteByte(0)
		buf.WriteString("test789") //password
		buf.WriteByte(0)
		buf.WriteString("4630212") //game version
		buf.WriteByte(0)

		steamKey := make([]byte, 2048)

		steamKey,_ = steamauth.CreateTicket()

		localsid := steamworks.GetSteamID()

		buf.WriteInt16(242)
		steamid64 := uint64(localsid)
		buf.WriteUint64(steamid64)

		if len(steamKey) > 0 {
			buf.WriteBytes(steamKey)
		}

		client.SendPacket(buf, false)
		return
	case 'B':
		if client.connectionStep < 3 {
			log.Println("Connected successfully")
			client.connectionStep = 3

			//client.channel.PrepareStreams()

			senddata := bitbuf.NewWriter(2048)

			senddata.WriteUnsignedBitInt32(6, 6)
			senddata.WriteByte(2)
			senddata.WriteInt32(-1)

			senddata.WriteUnsignedBitInt32(4, 8)
			senddata.WriteBytes([]byte("VModEnable 1"))
			senddata.WriteByte(0)
			senddata.WriteUnsignedBitInt32(4, 6)
			senddata.WriteString("vban 0 0 0 0")
			senddata.WriteByte(0)

			client.SendPacket(senddata, true)

			time.Sleep(3 * time.Second)
		}
	case 'I':
		return
	default:
		return
	}
}

func (client *Client) keepAlive() {
	go func() {
		for true {
			if client.connectionStep == 1 {
				buf := bitbuf.NewWriter(1000)
				buf.WriteByte(255)
				buf.WriteByte(255)
				buf.WriteByte(255)
				buf.WriteByte(255)
				buf.WriteByte('q')
				buf.WriteInt32(167679079)
				buf.WriteString("0000000000")
				buf.WriteByte(0)
				client.SendPacket(buf, false)
			}
			time.Sleep(1 * time.Second)
		}
	}()
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
		log.Println("type 6!")
		state,_ := packet.ReadUint8()
		serverCount,_ := packet.ReadInt32()

		log.Println(state)
		log.Println(serverCount)

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
			client.SendPacket(senddata, true)

			senddata = bitbuf.NewWriter(1000)
			senddata.WriteUnsignedBitInt32(0, 6)
			senddata.WriteUnsignedBitInt32(6, 6)
			senddata.WriteByte(state)
			senddata.WriteInt32(serverCount)
			client.SendPacket(senddata, true)
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

			client.SendPacket(senddata, true)

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
	c.keepAlive()

	return c
}