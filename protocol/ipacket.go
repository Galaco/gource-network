package protocol


type IPacket interface {
	Type() int
	ToBytes() []byte
}