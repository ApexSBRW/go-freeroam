package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	"time"

	"net/http"
	_ "net/http/pprof"

	"gitlab.com/rbs-nfsw/go-freeroam/internal"
)

func main() {
	i, _ := internal.Start(":9999")
	fmt.Println("Freeroam server running on port 9999")

	// pprof
	http.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		i.Lock()
		defer i.Unlock()
		out := make([]interface{}, 0)
		for addr, client := range i.Clients {
			pos := client.GetPos()
			slots := make(map[int]*string)
			for i, slot := range client.Slots {
				if slot == nil || slot.Client == nil {
					slots[i] = nil
				} else {
					addr := slot.Client.Addr.String()
					slots[i] = &addr
				}
			}
			out = append(out, map[string]interface{}{
				"addr":     addr,
				"ping":     client.Ping,
				"idle_for": math.Round(time.Now().Sub(client.LastPacket).Seconds() * 1000),
				"pos":      []float64{pos.X, pos.Y},
				"slots":    slots,
			})
		}
		b, _ := json.Marshal(out)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(200)
		rw.Write(b)
	})
	go http.ListenAndServe("localhost:6060", nil)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
