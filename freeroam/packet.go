package freeroam

import (
	"encoding/binary"
)

type CarPosPacket struct {
	packet []byte
	pos    Vector2D
}

func (p *CarPosPacket) Valid() bool {
	return p.packet != nil
}

func (p *CarPosPacket) Pos() Vector2D {
	return p.pos
}

func (p *CarPosPacket) Packet(timei uint16) []byte {
	time := make([]byte, 2)
	binary.BigEndian.PutUint16(time, timei)
	p.packet[0] = time[0]
	p.packet[1] = time[1]
	return p.packet
}

func (p *CarPosPacket) Update(packet []byte) {
	p.packet = packet
	flying := (packet[2] >> 3) & 1
	if flying == 1 {
		p.pos.X = int64(p.getX())
		p.pos.Y = int64(p.getY())
	}
}

func clone(a []byte) []byte {
	out := make([]byte, len(a))
	copy(out, a)
	return out
}

func (p *CarPosPacket) getY() uint32 {
	out := clone(p.packet[3:6])
	if p.isLowY() {
		out[2] = out[2] & 0xf8
	} else {
		out[2] = out[2] & 0xfc
	}
	return binary.BigEndian.Uint32([]byte{0x00, out[0], out[1], out[2]}) >> 2 & bitMask(17)
}

func (p *CarPosPacket) getX() uint32 {
	out := clone(p.packet[7:10])
	var shift uint
	if p.isLowY() {
		out[0] = out[0] & 0x7f
		out[2] = out[2] & 0xe0
		shift = 5
	} else {
		out[0] = out[0] & 0x3f
		out[2] = out[2] & 0xf0
		shift = 4
	}
	return binary.BigEndian.Uint32([]byte{0x00, out[0], out[1], out[2]}) >> shift & bitMask(18)
}

func (p *CarPosPacket) isLowY() bool {
	yHeader := binary.BigEndian.Uint16(p.packet[3:6])
	return yHeader <= 1941
}

func bitMask(n int) uint32 {
	out := 0x00
	for i := 0; i < n; i++ {
		out = out | (0x01 << uint(i))
	}
	return uint32(out)
}
