package udp

import (
	"github.com/BenLubar/steamworks"
	"github.com/BenLubar/steamworks/steamauth"
	"github.com/galaco/bitbuf"
	"github.com/galaco/gource-network/protocol"
	"log"
	"net"
)

type Client struct {
	conn net.Conn
}

// Disconnect from the current server
func (client *Client) Disconnect() {
	buf := bitbuf.NewWriter(64)
	buf.WriteInt8(1)
	buf.WriteString("Disconnect by User.")
	buf.WriteByte(0)
	client.Send(NewPacket(buf.Data()[:buf.BytesWritten()]))

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

	return NewPacket(buf[4:]), nil
}

// Perform initial connection back and forth
// with server
// See README.md for an explanation on how this works
func (client *Client) Reconnect() (int,error) {
	c := make(chan []byte)
	challengeFunc := func(step int, c chan []byte) {
		if step == 1 {
			buf := bitbuf.NewWriter(1000)
			buf.WriteByte(255)
			buf.WriteByte(255)
			buf.WriteByte(255)
			buf.WriteByte(255)
			buf.WriteByte('q')
			buf.WriteInt32(167679079)
			buf.WriteString("0000000000")
			buf.WriteByte(0)
			log.Println("B2B")
			log.Println(buf.Data()[:buf.BytesWritten()])
			client.Send(NewPacket(buf.Data()[:buf.BytesWritten()]))
		}
		if step == 2 {
			pkt,_ := client.Receive()
			c <- pkt.ToBytes()
		}
	}

	go challengeFunc(2, c)
	go challengeFunc(1, c)

	cData := <- c

	// PARSE RESPONSE
	rec := bitbuf.NewReader(cData)
	pktType,_ := rec.ReadByte()
	if pktType != 'A' {
		log.Fatalf("Bad packet type: %s", string(pktType))
	}

	rec.ReadInt32()
	serverchallenge,_ := rec.ReadInt32()
	ourchallenge,_ := rec.ReadInt32()

	connFunc := func(step int, c chan []byte) {
		if step == 1 {
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

			//steamuser->GetAuthSessionTicket(steamkey, STEAM_KEYSIZE, &keysize);
			steamKey,_ = steamauth.CreateTicket()

			localsid := steamworks.GetSteamID()

			buf.WriteInt16(242)
			steamid64 := uint64(localsid)
			buf.WriteUint64(steamid64)

			if len(steamKey) > 0 {
				buf.WriteBytes(steamKey)
			}

			client.Send(NewPacket(buf.Data()[:buf.BytesWritten()]))
		}
		if step == 2 {
			// RECEIVE PACKET
			pkt,_ := client.Receive()
			c <- pkt.ToBytes()
		}
	}

	go connFunc(2, c)
	go connFunc(1, c)

	response := <- c

	if response[0] == 'B' {
		log.Println("WE'VE CONNECTED!")
		return 4, nil
	} else {
		log.Printf("UHOH. Bad data: %s\n", string(response))
	}

	log.Println(response)


	return 4,nil
}

func NewClient() *Client {
	return &Client{}
}