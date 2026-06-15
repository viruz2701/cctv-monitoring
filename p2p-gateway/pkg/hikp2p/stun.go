package hikp2p

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const (
	stunMagicCookie = 0x2112A442
)

type StunResult struct {
	Address string
	Port    int
}

func RFC5389StunBind(server string, port int) (*StunResult, error) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(server), Port: port})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// Строим Binding Request
	transID := make([]byte, 12)
	rand.Read(transID)
	req := make([]byte, 20)
	binary.BigEndian.PutUint16(req[0:2], 0x0001) // Binding Request
	binary.BigEndian.PutUint16(req[2:4], 0)
	binary.BigEndian.PutUint32(req[4:8], stunMagicCookie)
	copy(req[8:20], transID)
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write(req); err != nil {
		return nil, err
	}
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	// Парсим XOR-MAPPED-ADDRESS
	if n < 20 {
		return nil, fmt.Errorf("short response")
	}
	msgType := binary.BigEndian.Uint16(buf[0:2])
	if msgType != 0x0101 {
		return nil, fmt.Errorf("not binding response")
	}
	msgLen := binary.BigEndian.Uint16(buf[2:4])
	offset := 20
	for offset < int(msgLen)+20 {
		if offset+4 > n {
			break
		}
		attrType := binary.BigEndian.Uint16(buf[offset : offset+2])
		attrLen := binary.BigEndian.Uint16(buf[offset+2 : offset+4])
		if attrType == 0x0020 { // XOR-MAPPED-ADDRESS
			family := buf[offset+5]
			if family == 0x01 { // IPv4
				port := binary.BigEndian.Uint16(buf[offset+6:offset+8]) ^ (stunMagicCookie >> 16)
				ip := binary.BigEndian.Uint32(buf[offset+8:offset+12]) ^ stunMagicCookie
				ipBytes := []byte{byte(ip >> 24), byte(ip >> 16), byte(ip >> 8), byte(ip)}
				return &StunResult{
					Address: net.IP(ipBytes).String(),
					Port:    int(port),
				}, nil
			}
		}
		offset += 4 + int(attrLen)
	}
	return nil, fmt.Errorf("no mapped address")
}
