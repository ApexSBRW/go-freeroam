package freeroam

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"time"
)

type clientPosSortInfo struct {
	Client *Client
	Length int
}

type clientPosSort []clientPosSortInfo

func (self clientPosSort) Len() int {
	return len(self)
}

func (self clientPosSort) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self clientPosSort) Less(i, j int) bool {
	return self[i].Length < self[j].Length
}

func newClient(cliTime []byte, addr *net.UDPAddr, conn *net.UDPConn, instance *Instance) *Client {
	slots := make(map[int]*slotInfo)
	for i := 0; i < 14; i++ {
		slots[i] = nil
	}
	c := &Client{
		Addr:       addr,
		conn:       conn,
		startTime:  time.Now(),
		cliTime:    clone(cliTime),
		seq:        0,
		Slots:      slots,
		LastPacket: time.Now(),
		instance:   instance,
	}
	return c
}

type Client struct {
	Addr       *net.UDPAddr
	conn       *net.UDPConn
	startTime  time.Time
	cliTime    []byte
	seq        uint16
	carPos     CarPosPacket
	chanInfo   []byte
	playerInfo []byte
	Slots      map[int]*slotInfo
	LastPacket time.Time
	Ping       int
	instance   *Instance
}

func (c Client) getTimeDiff() uint16 {
	return uint16(time.Now().Sub(c.startTime).Seconds() * 1000)
}

func (c *Client) getSeq() uint16 {
	out := c.seq
	c.seq++
	return out
}

func (c Client) replyHandshake() {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, c.getSeq())
	buf.WriteByte(0x01)
	binary.Write(buf, binary.BigEndian, c.getTimeDiff())
	buf.Write(c.cliTime)
	buf.Write([]byte{0x01, 0x01, 0x01, 0x01})
	c.conn.WriteToUDP(buf.Bytes(), c.Addr)
}

func (c Client) Active() bool {
	return time.Now().Sub(c.LastPacket).Seconds() < 5
}

func (c *Client) processPacket(packet []byte) {
	c.LastPacket = time.Now()
	if len(packet) == 58 && packet[2] == 0x06 {
		fmt.Printf("Re-Hello from %v\n", c.Addr.String())
		c.startTime = time.Now()
		c.cliTime = clone(packet[52:54])
		c.seq = 0
		slots := make(map[int]*slotInfo)
		for i := 0; i < 14; i++ {
			slots[i] = nil
		}
		c.Slots = slots
		c.carPos = CarPosPacket{}
		c.chanInfo = nil
		c.playerInfo = nil
		c.LastPacket = c.startTime
		return
	}

	if len(packet) <= 22 {
		return
	}

	data := packet[16 : len(packet)-5]
	reader := bytes.NewReader(data)
	for {
		ptype, err := reader.ReadByte()
		if err != nil {
			break
		}
		plen, _ := reader.ReadByte()
		//fmt.Printf("(FREEROAM) got packet 0x%x with len %d\n", ptype, plen)
		innerData := make([]byte, plen)
		reader.Read(innerData)
		//fmt.Println(hex.Dump(innerData))
		handled := false
		switch ptype {
		case 0x00:
			c.chanInfo = innerData
			handled = true
		case 0x01:
			c.playerInfo = innerData
			handled = true
		case 0x12:
			c.carPos.Update(innerData)
			time := binary.BigEndian.Uint16(innerData[0:2])
			c.Ping = int(c.getTimeDiff() - time)
			handled = true
		}
		if handled && c.isOk() {
			c.sendPlayerSlots()
		}
	}
}

func (self *Client) getClosestPlayers(clients []*Client) []*Client {
	closePlayers := make([]clientPosSortInfo, 0)
	for _, client := range clients {
		if !client.isOk() || client.Addr == self.Addr {
			continue
		}
		distance := self.GetPos().Sub(client.GetPos()).Abs().Length()
		//fmt.Printf("(FREEROAM) Distance between %s and %s is %v\n", self.Addr.String(), client.Addr.String(), distance)
		if distance <= 10000 {
			closePlayers = append(closePlayers, clientPosSortInfo{
				Length: int(distance),
				Client: client,
			})
		}
	}
	sort.Sort(clientPosSort(closePlayers))
	out := make([]*Client, 0)
	for _, p := range closePlayers {
		out = append(out, p.Client)
	}
	return out
}

func (self *Client) removeSlot(client *Client) {
	index := func() int {
		for i, c := range self.Slots {
			if c != nil && c.Client == client {
				return i
			}
		}
		return -1
	}()
	self.Slots[index] = nil
}

func (self *Client) addSlot(client *Client) {
	index := func() int {
		suitableSlots := make([]int, 0)
		for i, c := range self.Slots {
			if c == nil {
				suitableSlots = append(suitableSlots, i)
			}
		}
		sort.Ints(suitableSlots)

		if len(suitableSlots) == 0 {
			return len(self.Slots) - 1
		}

		return suitableSlots[0]

		//return len(suitableSlots)
	}()

	self.Slots[index] = &slotInfo{
		JustAdded: true,
		Client:    client,
	}
}

func (self *Client) recalculateSlots(clients []*Client) {
	players := self.getClosestPlayers(clients)
	oldPlayers := make([]*Client, 0)
	for _, v := range self.Slots {
		if v != nil {
			oldPlayers = append(oldPlayers, v.Client)
		}
	}
	diff := ArrayDiff(oldPlayers, players)
	for _, c := range diff.Removed {
		self.removeSlot(c)
	}
	for _, c := range diff.Added {
		self.addSlot(c)
	}
}

func (self *Client) sendPlayerSlots() {
	clients := make([]*Client, len(self.instance.Clients))
	{
		i := 0
		for _, c := range self.instance.Clients {
			clients[i] = c
			i++
		}
	}
	self.recalculateSlots(clients)
	buf := self.instance.buffers[0]
	buf.Reset()
	seq := self.getSeq()
	binary.Write(buf, binary.BigEndian, seq)
	buf.WriteByte(0x02)
	binary.Write(buf, binary.BigEndian, self.getTimeDiff())
	buf.Write(self.cliTime)
	binary.Write(buf, binary.BigEndian, seq)
	buf.Write([]byte{0xff, 0xff, 0x00})
	for i := 0; i < 14; i++ {
		slot := self.Slots[i]
		//fmt.Printf("%d: %#v\n", i, slot)
		if slot == nil {
			buf.Write([]byte{0xff, 0xff})
		} else {
			if slot.JustAdded {
				buf.Write(slot.Client.getFullSlotPacket(self.getTimeDiff() - 15))
				slot.JustAdded = false
			} else {
				buf.Write(slot.Client.getFullPosPacket(self.getTimeDiff() - 15))
			}
		}
	}
	buf.Write([]byte{0x01, 0x01, 0x01, 0x01})
	self.conn.WriteToUDP(buf.Bytes(), self.Addr)
}

func (c Client) GetPos() Vector2D {
	return c.carPos.Pos()
}

func (c Client) isOk() bool {
	return c.chanInfo != nil && c.playerInfo != nil && c.carPos.Valid()
}

func (c Client) getFullPosPacket(time uint16) []byte {
	buf := c.instance.buffers[1]
	buf.Reset()
	buf.WriteByte(0x00) // Slot start
	buf.WriteByte(0x12) // Type
	buf.WriteByte(0x1a) // Size
	buf.Write(c.carPos.Packet(time))
	buf.WriteByte(0xff) // Slot end
	return buf.Bytes()
}

func (c Client) getFullSlotPacket(time uint16) []byte {
	buf := c.instance.buffers[1]
	buf.Reset()
	buf.WriteByte(0x00) // Slot start
	buf.WriteByte(0x00) // Type
	buf.WriteByte(0x22) // Size
	buf.Write(c.chanInfo)
	buf.WriteByte(0x01) // Type
	buf.WriteByte(0x41) // Size
	buf.Write(c.playerInfo)
	buf.WriteByte(0x12) // Type
	buf.WriteByte(0x1a) // Size
	buf.Write(c.carPos.Packet(time))
	buf.WriteByte(0xff) // Slot end
	return buf.Bytes()
}
