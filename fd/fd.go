package failuredetector

import (
	"fmt"
	"strconv"
	"time"

	ld "github.com/abolfazl-talebzadeh/goMods/ld"
)

// EvtFailureDetector represents a Eventually Perfect Failure Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type EvtFailureDetector struct {
	id        int          // the id of this node
	nodeIDs   []int        // node ids for every node in cluster
	alive     map[int]bool // map of node ids considered alive
	suspected map[int]bool // map of node ids  considered suspected

	sr SuspectRestorer // Provided SuspectRestorer implementation

	delay         time.Duration // the current delay for the timeout procedure
	delta         time.Duration // the delta value to be used when increasing delay
	timeoutSignal *time.Ticker  // the timeout procedure ticker

	hbSend chan<- Heartbeat // channel for sending outgoing heartbeat messages
	hbIn   chan Heartbeat   // channel for receiving incoming heartbeat messages
	stop   chan struct{}    // channel for signaling a stop request to the main run loop

	testingHook func() // DO NOT REMOVE THIS LINE. A no-op when not testing.
}

// NewEvtFailureDetector returns a new Eventual Failure Detector. It takes the
// following arguments:
//
// id: The id of the node running this instance of the failure detector.
//
// nodeIDs: A list of ids for every node in the cluster (including the node
// running this instance of the failure detector).
//
// ld: A leader detector implementing the SuspectRestorer interface.
//
// delta: The initial value for the timeout interval. Also the value to be used
// when increasing delay.
//
// hbSend: A send only channel used to send heartbeats to other nodes.
func NewEvtFailureDetector(id int, nodeIDs []int, sr SuspectRestorer, delta time.Duration, hbSend chan<- Heartbeat) *EvtFailureDetector {
	suspected := make(map[int]bool)
	alive := make(map[int]bool)
	for _, in := range nodeIDs {
		alive[in] = true
	}
	return &EvtFailureDetector{
		id:        id,
		nodeIDs:   nodeIDs,
		alive:     alive,
		suspected: suspected,

		sr: sr,

		delay: delta,
		delta: delta,

		hbSend: hbSend,
		hbIn:   make(chan Heartbeat, 8),
		stop:   make(chan struct{}),

		testingHook: func() {}, // DO NOT REMOVE THIS LINE. A no-op when not testing.
	}
}

// Start starts e's main run loop as a separate goroutine. The main run loop
// handles incoming heartbeat requests and responses. The loop also trigger e's
// timeout procedure at an interval corresponding to e's internal delay
// duration variable.
func (e *EvtFailureDetector) Start() {
	e.timeoutSignal = time.NewTicker(e.delay)
	go func(timeout <-chan time.Time) {
		for {
			e.testingHook() // DO NOT REMOVE THIS LINE. A no-op when not testing.
			select {
			case heartBeat1 := <-e.hbIn:
				if heartBeat1.Request {
					var _from int
					var _To int
					_from = e.id
					_To = heartBeat1.From
					e.Send(Heartbeat{From: _from, To: _To, Request: false})
				} else {
					e.alive[heartBeat1.From] = true
				}
			case <-timeout:
				e.timeout()
			case <-e.stop:
				fmt.Println("fd stoped")
				return
			}
		}
	}(e.timeoutSignal.C)
}

func (e *EvtFailureDetector) Send(hbTemp Heartbeat) {
	e.hbSend <- hbTemp
}

// DeliverHeartbeat delivers heartbeat hb to failure detector e.
func (e *EvtFailureDetector) DeliverHeartbeat(hb Heartbeat) {
	e.hbIn <- hb
}

// Stop stops e's main run loop.
func (e *EvtFailureDetector) Stop() {
	e.stop <- struct{}{}
}

// Internal: timeout runs e's timeout procedure.
func (e *EvtFailureDetector) timeout() {
	if matchIntersections(e.alive, e.suspected) {
		e.delay += e.delta
	}
	for _, index := range e.nodeIDs {
		if index == e.id {
			e.suspected[index] = false
			continue
		}
		if !e.alive[index] && !e.suspected[index] {
			e.suspected[index] = true
			e.sr.Suspect(index)
		} else if e.alive[index] && e.suspected[index] {
			delete(e.suspected, index)
			e.sr.Restore(index)
		}
		heartBeat := Heartbeat{From: e.id, To: index, Request: true}
		e.hbSend <- heartBeat
	}
	e.alive = make(map[int]bool)
	e.timeoutSignal = time.NewTicker(e.delay)
}
func matchIntersections(alive, suspected map[int]bool) bool {
	for id, flag := range alive {
		if flag && suspected[id] {
			return true
		}
	}
	return false
}

func (e *EvtFailureDetector) AddNewNode(nodeID int) {
	existed := false
	for _, nID := range e.nodeIDs {
		if nID == nodeID {
			existed = true
		}
	}
	if existed {
		e.suspected[nodeID] = false
		e.alive[nodeID] = true
	} else {
		e.nodeIDs = append(e.nodeIDs, nodeID)
		e.alive[nodeID] = true
	}
}

func (e *EvtFailureDetector) UpdateSR(m *ld.MonLeaderDetector) {
	for _, nodeID := range e.nodeIDs {
		if e.suspected[nodeID] {
			m.Suspect(nodeID)
		} else {
			m.Restore(nodeID)
		}
	}
}

func (e *EvtFailureDetector) SetID(hash int) {
	e.id = hash
}

func (e *EvtFailureDetector) NodeList() []string {
	var stringNodes []string
	for _, nodes := range e.nodeIDs {
		stringNodes = append(stringNodes, strconv.Itoa(nodes))
	}
	return stringNodes
}

func (e *EvtFailureDetector) ShowSuspended() map[int]bool {
	return e.suspected
}

func (e *EvtFailureDetector) ShowAlive() map[int]bool {
	return e.alive
}

func (e *EvtFailureDetector) ShowNodes() []int {
	return e.nodeIDs
}
