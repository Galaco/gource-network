package protocol


type IPacket interface {
	ToBytes() []byte
}