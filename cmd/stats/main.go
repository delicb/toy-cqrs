package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/nats-io/nats.go"
)

func main() {
	natsConn, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		panic(err)
	}

	monitor := &statsMonitor{}

	_, err = natsConn.Subscribe("command.>", monitor.onMessage)
	if err != nil {
		panic(err)
	}
	_, err = natsConn.Subscribe("event.>", monitor.onMessage)
	if err != nil {
		panic(err)
	}

	http.Handle("/", monitor)
	log.Fatal(http.ListenAndServe("0.0.0.0:8010", nil))
}

type statsMonitor struct {
	cmdCount         uint32
	eventsCount      uint32
	errorEventsCount uint32
}

func (p *statsMonitor) onMessage(msg *nats.Msg) {
	sub := msg.Subject
	if strings.HasPrefix(sub, "command") {
		atomic.AddUint32(&p.cmdCount, 1)
	} else if strings.HasPrefix(sub, "event") {
		atomic.AddUint32(&p.eventsCount, 1)
	}

	if strings.HasSuffix(sub, "error") {
		atomic.AddUint32(&p.errorEventsCount, 1)
	}
}

func (p *statsMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "total commands: %d\n", p.cmdCount)
	fmt.Fprintf(w, "total events: %d\n", p.eventsCount)
	fmt.Fprintf(w, "total error events: %d\n", p.errorEventsCount)
	if p.eventsCount > 0 {
		fmt.Fprintf(w, "total errors percentage: %d\n", p.errorEventsCount/p.eventsCount*100)
	}
	w.WriteHeader(http.StatusOK)
}
