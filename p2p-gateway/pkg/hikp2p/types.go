package hikp2p

import "time"

type Credentials struct {
	Username string
	Password string
}

type Session struct {
	SessionID    string
	ApiDomain    string
	ExpiresAt    time.Time
	RefreshToken string
}

type P2PServer struct {
	Host string
	Port int
}

type P2PSecret struct {
	Key       []byte
	SaltIndex int
	SaltVer   int
	Servers   []P2PServer
}

type Device struct {
	Serial     string
	Name       string
	IP         string
	Port       int
	LocalIP    string
	LocalPort  int
	P2PVersion int
}

type P2PConfig struct {
	DeviceSerial  string
	DeviceIP      string
	DevicePort    int
	P2PServers    []P2PServer
	P2PKey        []byte
	P2PLinkKey    []byte
	P2PKeyVersion int
	P2PKeySaltIdx int
	P2PKeySaltVer int
	SessionToken  string
	UserID        string
	ClientID      uint32 // изменено с int на uint32
	ChannelNo     int
	StreamType    int    // 1 – main, 2 – sub
	StartTime     string // опционально, для playback
	StopTime      string // опционально, для playback
}
