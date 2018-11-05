package udp

const PacketHeaderFlagQuery = -1
const PacketHeaderFlagSplit = -2
const PacketHeaderFlagCompressed = -3

const TypeNOP = 0
const TypeDisconnect = 1
const TypeNetFile = 2
const TypeNetTick = 3
const TypeNetStringCommand = 4
const TypeNetSetConVar = 5
const TypeNetSignonState = 6
const TypeSvcPrint = 7
const TypeSvcServerInfo = 8
const TypeSvcClassInfo = 10
const TypeSvcSetPause = 11
const TypeSvcCreateStringTable = 12
const TypeSvcUpdateStringTable = 13
const TypeSvcVoiceInit = 14
const TypeSvcVoiceData = 15
const TypeSvcSounds = 17
const TypeSvcSetView = 18
const TypeSvcFixAngle = 19
const TypeSvcCrosshairAngle = 20
const TypeSvcBSPDecal = 21
const TypeSvcUserMessage = 23
const TypeSvcEntityMessage = 24
const TypeSvcGameEvent = 25
const TypeSvcPacketEntities = 26
const TypeSvcTempEntities = 27
const TypeSvcPrefetch = 28
const TypeSvcGameEventList = 30
const TypeSvcGetCvarValue = 31

type Packet struct {
	body []byte
}

func (packet *Packet) ToBytes() []byte {
	return packet.body
}

func NewPacket(body []byte) *Packet {

	return &Packet{
		body: body,
	}
}