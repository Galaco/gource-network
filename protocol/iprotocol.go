package protocol

type IProtocol interface {
	Disconnect()
	Connect(host string, port string) error
	Send(packet IPacket) error
	Receive() (IPacket,error)
	Reconnect() (int,error)
}
