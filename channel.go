package network

import (
	"github.com/galaco/bitbuf"
	"github.com/galaco/gource-network/internal"
	"log"
)

const checksumPackets = true

const packetFlagReliable = 1 << 0
const packetFlagCompressed = 1 << 1
const packetFlagEncrypted = 1 << 2
const packetFlagSplit = 1 << 3
const packetFlagChoked = 1 << 4
const packetFlagChallenge = 1 << 5
const packetFlagTables = 1 << 10

const maxStreams = 2

type Channel struct {
	droppedPackets          int32
	inSequenceNr            int32
	outSequenceNrAck        int32
	outReliableState        int32
	inReliableState         int32
	outSequenceNr           int32
	signOnState             int32
	serverCount             int32
	challengeNR             uint32
	streamContainsChallenge bool
}


func (channel *Channel) ReadPacketHeader(buf *bitbuf.Reader) int32 {
	sequence,_ := buf.ReadInt32()
	sequenceAck,_ := buf.ReadInt32()
	bflags,_ := buf.ReadByte()
	flags := int32(bflags)

	//validate checksum
	if checksumPackets == true {
		checkSum,_ := buf.ReadUint16()
		offset := buf.BitsRead() >> 3
		checkSumBytes := buf.Data()[:(buf.Size() >> 3) - offset]

		usDataCheckSum := internal.CRC32(checkSumBytes)

		if usDataCheckSum != checkSum {
			// bad packet.
		}
	}

	buf.ReadByte() //relState
	numChoked := uint8(0)
	if flags & packetFlagChoked != 0 {
		numChoked,_ = buf.ReadUint8()
	}
	if flags & packetFlagChallenge != 0 {
		challenge,_ := buf.ReadUint32()
		challenge = 100
		if channel.challengeNR == 0 {
			channel.challengeNR = challenge
		}

		if challenge != channel.challengeNR {
			log.Printf("Bad challenge: %d\n", channel)
			return -1
		}
		channel.streamContainsChallenge = true
	} else {
		if channel.streamContainsChallenge == true {
			return -1
		}
	}

	//discard duplicate or out of date packet
	if sequence <= channel.inSequenceNr {
		//duplicate or old packet
		//
	}

	channel.droppedPackets = sequence - (channel.inSequenceNr + int32(numChoked) + 1)

	// There are dropped packets
	if channel.droppedPackets > 0 {
		// Do something about dropped packets?
	}

	channel.inSequenceNr = sequence
	channel.outSequenceNrAck = sequenceAck

	// Update waiting list status
	//for i := 0; i < maxStreams; i++ {
	//	CheckWaitingList(i)
	//}


	if sequence == 0x36 {
		flags |= packetFlagTables
	}

	return flags
}

func NewChannel() *Channel {
	return &Channel {
		droppedPackets:          0,
		inSequenceNr:            0,
		outSequenceNrAck:        0,
		outReliableState:        0,
		inReliableState:     	 0,
		outSequenceNr:			 1,
		signOnState:             2,
		serverCount:             -1,
		challengeNR:             0,
		streamContainsChallenge: false,
	}
}