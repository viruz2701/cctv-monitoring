package hikp2p

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

type SessionState string

const (
	StateIdle      SessionState = "idle"
	StatePunching  SessionState = "punching"
	StateSetup     SessionState = "setup"
	StateStreaming SessionState = "streaming"
	StateStopped   SessionState = "stopped"
)

type P2PSession struct {
	config P2PConfig
	conn   *net.UDPConn
	state  SessionState
	mu     sync.Mutex

	// P2P state
	seqNum          uint32
	sessionKey      string
	deviceAddr      *net.UDPAddr
	deviceSessionId uint32

	// SRT‑like state
	srtPeerSocketId uint32
	srtLastAckSeq   uint32
	srtAckTimer     *time.Timer
	srtDataCount    uint32

	// Video pipeline
	videoReader *io.PipeReader
	videoWriter *io.PipeWriter

	// Control
	done      chan struct{}
	closeOnce sync.Once
}

func NewP2PSession(cfg P2PConfig) (*P2PSession, error) {
	pr, pw := io.Pipe()
	return &P2PSession{
		config:      cfg,
		state:       StateIdle,
		sessionKey:  generateSessionKey(cfg.DeviceSerial, cfg.ChannelNo),
		videoReader: pr,
		videoWriter: pw,
		done:        make(chan struct{}),
	}, nil
}

// generateSessionKey – как в оригинальном клиенте: base64(serial)+channel+timestamp+random
func generateSessionKey(serial string, channel int) string {
	b64 := base64StdEncode([]byte(serial))
	now := time.Now()
	ts := fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	randPart := make([]byte, 3)
	rand.Read(randPart)
	return fmt.Sprintf("%s%d%s%x", b64, channel, ts, randPart)
}

func base64StdEncode(src []byte) string {
	const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	dst := make([]byte, 4*((len(src)+2)/3))
	si, di := 0, 0
	for si < len(src) {
		val := uint(src[si]) << 16
		if si+1 < len(src) {
			val |= uint(src[si+1]) << 8
		}
		if si+2 < len(src) {
			val |= uint(src[si+2])
		}
		dst[di] = b64[(val>>18)&0x3F]
		dst[di+1] = b64[(val>>12)&0x3F]
		if si+1 < len(src) {
			dst[di+2] = b64[(val>>6)&0x3F]
		} else {
			dst[di+2] = '='
		}
		if si+2 < len(src) {
			dst[di+3] = b64[val&0x3F]
		} else {
			dst[di+3] = '='
		}
		si += 3
		di += 4
	}
	return string(dst)
}

func (s *P2PSession) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateIdle {
		return fmt.Errorf("session already started")
	}
	s.state = StatePunching

	// 1. Создаём UDP сокет
	localAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return err
	}
	s.conn = conn

	// 2. P2P_SETUP на все сервера
	if err := s.sendP2PSetup(); err != nil {
		s.conn.Close()
		return err
	}

	// 3. Ожидаем hole‑punch от устройства (0x0C00)
	go s.readLoop()
	if err := s.waitForPunch(10 * time.Second); err != nil {
		s.conn.Close()
		return err
	}
	s.state = StateSetup

	// 4. PLAY_REQUEST (прямой + relay)
	if err := s.sendPlayRequest(); err != nil {
		s.conn.Close()
		return err
	}

	// 5. Ждём установки SRT‑подобной сессии (0x8000 handshake)
	if err := s.waitForDataSession(15 * time.Second); err != nil {
		s.conn.Close()
		return err
	}

	s.state = StateStreaming
	go s.keepAliveLoop()
	return nil
}

func (s *P2PSession) readLoop() {
	buf := make([]byte, 65535)
	for {
		select {
		case <-s.done:
			return
		default:
		}
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		data := buf[:n]
		// Определяем тип пакета
		if len(data) >= 2 && (data[0]>>4) == 0xE {
			// V3 сообщение от P2P сервера
			s.handleV3Message(data, addr)
		} else if len(data) >= 16 && (data[0]&0x80) == 0 {
			// SRT‑подобный data packet (F=0)
			s.handleSrtData(data)
		} else if len(data) >= 2 && data[0] == 0x80 && data[1] == 0x00 {
			// SRT handshake / control
			s.handleSrtControl(data)
		} else {
			// Неизвестный – игнорируем
		}
	}
}

// --- P2P_SETUP ---
func (s *P2PSession) sendP2PSetup() error {
	// Формируем тело атрибутов (аналог TypeScript buildP2PSetupRequest)
	attrs := []V3Attribute{
		{Tag: 0x05, Value: []byte(s.sessionKey)},
		{Tag: 0x06, Value: []byte(s.config.UserID)},
		{Tag: 0x00, Value: []byte(s.config.DeviceSerial)},
		{Tag: 0x04, Value: []byte{0x03}}, // версия протокола 3
	}
	// Композитный атрибут 0xFF (транспортная информация)
	localAddr := s.conn.LocalAddr().(*net.UDPAddr)
	localAddrStr := fmt.Sprintf("%s:%d", localAddr.IP.String(), localAddr.Port)
	transforParts := []V3Attribute{
		{Tag: 0x71, Value: []byte{byte(s.config.StreamType)}}, // busType
		{Tag: 0x72, Value: []byte{0x03}},
		{Tag: 0x75, Value: []byte{0x01}},
		{Tag: 0x7f, Value: []byte{0x0a}},
		{Tag: 0x74, Value: []byte(localAddrStr)},
		{Tag: 0x8c, Value: uint32ToBE(s.config.ClientID)},
	}
	transforData, _ := encodeAttributes(transforParts, true)
	attrs = append(attrs, V3Attribute{Tag: 0xFF, Value: transforData})

	msg := V3Message{
		MsgType:    0x0B02,
		Seq:        s.nextSeq(),
		Encrypt:    true,
		SaltVer:    s.config.P2PKeySaltVer,
		SaltIdx:    s.config.P2PKeySaltIdx,
		Is2BLen:    true,
		Attributes: attrs,
	}
	packet, err := EncodeV3Message(msg, s.config.P2PKey)
	if err != nil {
		return err
	}
	for _, srv := range s.config.P2PServers {
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", srv.Host, srv.Port))
		s.conn.WriteToUDP(packet, addr)
	}
	return nil
}

func (s *P2PSession) waitForPunch(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("punch timeout")
		case <-s.done:
			return fmt.Errorf("session closed")
		}
	}
}

// --- PLAY_REQUEST ---
func (s *P2PSession) sendPlayRequest() error {
	// Внутреннее тело PLAY_REQUEST (TLV)
	bodyAttrs := []V3Attribute{
		{Tag: 0x76, Value: []byte{byte(s.config.StreamType)}}, // busType 1=live
		{Tag: 0x05, Value: []byte(s.sessionKey)},
		{Tag: 0x78, Value: []byte{byte(s.config.StreamType)}}, // streamType
		{Tag: 0x77, Value: uint16ToBE(uint16(s.config.ChannelNo))},
		{Tag: 0x7e, Value: uint32ToBE(uint32(time.Now().Unix()))}, // streamSession
		{Tag: 0x7d, Value: uint32ToBE(180)},                       // timeout
		{Tag: 0x7a, Value: []byte(s.config.StartTime)},            // если playback
		{Tag: 0x7b, Value: []byte(s.config.StopTime)},
		{Tag: 0x83, Value: []byte(s.config.DeviceSerial)},
	}
	innerBody, _ := encodeAttributes(bodyAttrs, true)

	// Шифруем внутреннее тело P2PLinkKey (первые 16 байт)
	encInner, _ := aes128CBCEncrypt(innerBody, s.config.P2PLinkKey[:16])

	// Expand header (48 байт)
	expandAttrs := []V3Attribute{
		{Tag: 0x00, Value: uint16ToBE(uint16(s.config.P2PKeyVersion))},
		{Tag: 0x01, Value: []byte(s.config.UserID)},
		{Tag: 0x02, Value: uint32ToBE(uint32(s.config.ClientID))},
		{Tag: 0x03, Value: uint16ToBE(uint16(s.config.ChannelNo))},
	}
	expandData, _ := encodeAttributes(expandAttrs, true)

	// Внутреннее V3 сообщение (PLAY_REQUEST)
	innerV3 := V3Message{
		MsgType:   0x0C02,
		Seq:       s.nextSeq(),
		Encrypt:   false, // уже зашифровано
		ExpandHdr: true,
		Is2BLen:   true,
		Attributes: []V3Attribute{
			{Tag: 0x07, Value: encInner},
		},
	}
	innerPacket, err := EncodeV3Message(innerV3, nil)
	if err != nil {
		return err
	}
	// Вставляем expand header после заголовка (12 байт + expandData)
	innerWithExpand := make([]byte, 12+len(expandData)+len(innerPacket[12:]))
	copy(innerWithExpand[0:12], innerPacket[0:12])
	copy(innerWithExpand[12:12+len(expandData)], expandData)
	copy(innerWithExpand[12+len(expandData):], innerPacket[12:])
	// Пересчитываем CRC
	innerWithExpand[11] = 0
	innerWithExpand[11] = crc8(innerWithExpand)

	// Внешний TRANSFOR_DATA (0x0B04)
	outerBody := []byte{0x00, byte(len(s.config.DeviceSerial))}
	outerBody = append(outerBody, []byte(s.config.DeviceSerial)...)
	outerBody = append(outerBody, 0x07, 0x00, 0x00) // tag 0x07 + длина (заполним позже)
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(innerWithExpand)))
	outerBody = append(outerBody, lenBuf...)
	outerBody = append(outerBody, innerWithExpand...)

	encOuter, _ := aes128CBCEncrypt(outerBody, s.config.P2PKey[:16])
	outerMsg := V3Message{
		MsgType: 0x0B04,
		Seq:     s.nextSeq(),
		Encrypt: true,
		SaltVer: s.config.P2PKeySaltVer,
		SaltIdx: s.config.P2PKeySaltIdx,
		Is2BLen: true,
		Attributes: []V3Attribute{
			{Tag: 0x07, Value: encOuter},
		},
	}
	packet, err := EncodeV3Message(outerMsg, s.config.P2PKey)
	if err != nil {
		return err
	}

	// Отправляем на все P2P сервера и напрямую устройству
	for _, srv := range s.config.P2PServers {
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", srv.Host, srv.Port))
		s.conn.WriteToUDP(packet, addr)
	}
	if s.deviceAddr != nil {
		s.conn.WriteToUDP(packet, s.deviceAddr)
	}
	return nil
}

// --- SRT-подобный протокол ---
func (s *P2PSession) handleSrtControl(data []byte) {
	if len(data) < 16 {
		return
	}
	typ := binary.BigEndian.Uint16(data[0:2])
	if typ != 0x8000 {
		return
	}
	handshakeType := binary.BigEndian.Uint32(data[36:40])
	peerSocketId := binary.BigEndian.Uint32(data[12:16])
	if handshakeType == 1 { // INDUCTION
		// Ответить induction response
		resp := make([]byte, 64)
		copy(resp[0:16], data[0:16])                        // копируем заголовок
		binary.BigEndian.PutUint32(resp[36:40], 1)          // HS type = 1 (response)
		binary.BigEndian.PutUint32(resp[40:44], 0x12345678) // наш socket id
		binary.BigEndian.PutUint32(resp[44:48], uint32(time.Now().Unix()))
		s.conn.WriteToUDP(resp, s.deviceAddr)
		s.srtPeerSocketId = peerSocketId
	} else if handshakeType == 0xFFFFFFFF { // CONCLUSION
		// Ответить conclusion response
		resp := make([]byte, 64)
		copy(resp[0:16], data[0:16])
		binary.BigEndian.PutUint32(resp[36:40], 0xFFFFFFFF)
		binary.BigEndian.PutUint32(resp[40:44], s.srtPeerSocketId)
		s.conn.WriteToUDP(resp, s.deviceAddr)
		// Запускаем таймер ACK
		s.startAckTimer()
	}
}

func (s *P2PSession) handleSrtData(data []byte) {
	if len(data) < 20 {
		return
	}
	seq := binary.BigEndian.Uint32(data[0:4]) & 0x7FFFFFFF
	payload := data[16:]
	if len(payload) < 2 {
		return
	}
	// Проверяем тип полезной нагрузки – 0x8060/0x8050/0x8051
	ptype := binary.BigEndian.Uint16(payload[0:2])
	if ptype != 0x8060 && ptype != 0x8050 && ptype != 0x8051 {
		return
	}
	// Извлекаем H.265 NAL
	nal := ExtractNALU(payload)
	if nal != nil {
		// Отправляем в pipeline с Annex B start code
		s.videoWriter.Write(append([]byte{0, 0, 0, 1}, nal...))
	}
	s.srtLastAckSeq = seq
	s.srtDataCount++
}

func (s *P2PSession) startAckTimer() {
	if s.srtAckTimer != nil {
		s.srtAckTimer.Stop()
	}
	s.srtAckTimer = time.AfterFunc(10*time.Millisecond, func() {
		s.sendSrtAck()
	})
}

func (s *P2PSession) sendSrtAck() {
	if s.deviceAddr == nil || s.srtPeerSocketId == 0 {
		return
	}
	ack := make([]byte, 44)
	binary.BigEndian.PutUint16(ack[0:2], 0x8002) // ACK
	binary.BigEndian.PutUint32(ack[12:16], s.srtPeerSocketId)
	binary.BigEndian.PutUint32(ack[16:20], s.srtLastAckSeq+1)
	binary.BigEndian.PutUint32(ack[20:24], 8000) // RTT
	s.conn.WriteToUDP(ack, s.deviceAddr)
}

func (s *P2PSession) keepAliveLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.sendKeepalive()
		case <-s.done:
			return
		}
	}
}

func (s *P2PSession) sendKeepalive() {
	if s.deviceAddr == nil {
		return
	}
	ka := make([]byte, 20)
	binary.BigEndian.PutUint16(ka[0:2], 0x8001)
	s.conn.WriteToUDP(ka, s.deviceAddr)
}

// --- V3 обработчик ---
func (s *P2PSession) handleV3Message(data []byte, addr *net.UDPAddr) {
	msg, err := DecodeV3Message(data, s.config.P2PKey)
	if err != nil {
		return
	}
	switch msg.MsgType {
	case 0x0B03: // P2P_SETUP response
		// Извлекаем tag 0xFF -> sub‑tag 0x74 (IP:PORT устройства)
		for _, attr := range msg.Attributes {
			if attr.Tag == 0xFF {
				sub, _ := decodeAttributes(attr.Value, true)
				for _, subAttr := range sub {
					if subAttr.Tag == 0x74 {
						addrStr := string(subAttr.Value)
						host, portStr, _ := net.SplitHostPort(addrStr)
						port, _ := strconv.Atoi(portStr)
						s.deviceAddr = &net.UDPAddr{IP: net.ParseIP(host), Port: port}
						s.deviceSessionId = binary.BigEndian.Uint32(subAttr.Value[12:16])
						break
					}
				}
			}
		}
	case 0x0B05: // TRANSFOR_DATA response – может содержать подтверждение PLAY_REQUEST
		// Анализируем tag 0x02 – код ошибки
		for _, attr := range msg.Attributes {
			if attr.Tag == 0x02 && len(attr.Value) == 4 {
				errCode := binary.BigEndian.Uint32(attr.Value)
				if errCode != 0 {
					fmt.Printf("P2P error: %d\n", errCode)
				}
			}
		}
	}
}

func (s *P2PSession) waitForDataSession(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("data session timeout")
		case <-s.done:
			return fmt.Errorf("session closed")
		default:
			if s.deviceAddr != nil && s.srtPeerSocketId != 0 {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *P2PSession) Stop() error {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.conn != nil {
			s.conn.Close()
		}
		s.videoWriter.Close()
	})
	return nil
}

func (s *P2PSession) VideoReader() io.ReadCloser {
	return s.videoReader
}

func (s *P2PSession) nextSeq() uint32 {
	s.seqNum++
	return s.seqNum
}

// --- Вспомогательные функции для сериализации чисел ---
func uint16ToBE(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}
func uint32ToBE(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}
