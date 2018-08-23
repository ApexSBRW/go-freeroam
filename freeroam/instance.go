package freeroam

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"
)

func Start(addrStr string) (*Instance, error) {
	addr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	i := NewInstance(listener)
	go i.RunPacketRead()
	go i.RunTimer()
	return i, nil
}

func NewInstance(listener *net.UDPConn) *Instance {
	return &Instance{
		listener: listener,
		Clients:  make(map[string]*Client),
		udpbuf:   make([]byte, 1024),
		buffers: []*bytes.Buffer{
			new(bytes.Buffer),
			new(bytes.Buffer),
		},
	}
}

type Instance struct {
	sync.Mutex
	listener *net.UDPConn
	Clients  map[string]*Client
	udpbuf   []byte
	buffers  []*bytes.Buffer
}

func (i *Instance) RunPacketRead() {
	for {
		addr, data := i.readPacket()
		i.Lock()
		client, ok := i.Clients[addr.String()]
		if !ok {
			if len(data) == 58 && data[2] == 0x06 {
				// Handshake
				fmt.Printf("New client from %v\n", addr.String())
				i.Clients[addr.String()] = newClient(data[52:54], addr, i.listener, i)
				i.Clients[addr.String()].replyHandshake()
			} else {
				// Something other
				i.Unlock()
				continue
			}
		} else {
			client.processPacket(data)
		}
		i.Unlock()
	}
}

func (i *Instance) RunTimer() {
	timer := time.Tick(1 * time.Second)
	for {
		i.Lock()
		remove := make([]string, 0)
		{
			j := 0
			for k, client := range i.Clients {
				if !client.Active() {
					remove = append(remove, k)
					fmt.Printf("Removing inactive client %v\n", client.Addr.String())
				}
				j++
			}
		}
		for _, k := range remove {
			delete(i.Clients, k)
		}
		i.Unlock()
		<-timer
	}
}

func (i *Instance) readPacket() (*net.UDPAddr, []byte) {
	len, addr, _ := i.listener.ReadFromUDP(i.udpbuf)
	return addr, i.udpbuf[:len]
}
