package network

// Provides necesarry information to
// identify as a nosteam client to a server
type ClientInfo struct {
	ClientChallenge uint32
	Name string
	GameVersion string
	FakeSteamId int64
}
