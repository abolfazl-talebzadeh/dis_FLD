package leaderdetector

import "fmt"

// A MonLeaderDetector represents a Monarchical Eventual Leader Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type MonLeaderDetector struct {
	NodeID      []int //the slice holding the IDS for all nodes
	suspected   map[int]bool
	subscribers []chan int
	//suspectHistory map[int]int
	leader int
	alive  bool
}

// NewMonLeaderDetector returns a new Monarchical Eventual Leader Detector
// given a list of node ids.
func NewMonLeaderDetector(nodeIDs []int) *MonLeaderDetector {
	var suspected = make(map[int]bool)
	var subscriber []chan int
	for _, index := range nodeIDs {
		suspected[index] = false
	}

	m := &MonLeaderDetector{NodeID: nodeIDs, suspected: suspected, subscribers: subscriber, leader: UnknownID, alive: true}
	fmt.Println("------>", m)
	return m
}

// Leader returns the current leader. Leader will return UnknownID if all nodes
// are suspected.
func (m *MonLeaderDetector) Leader() int {
	tempLeader := UnknownID
	if !m.alive {
		return tempLeader
	}
	//for index := len(m.NodeID) - 1; index >= 0; index-- {
	for index, id := range m.NodeID {
		if !m.isSuspected(m.NodeID[index]) && m.NodeID[index] >= 0 && m.NodeID[index] > tempLeader {
			tempLeader = id
		}
	}
	return tempLeader
}

// Suspect instructs the leader detector to consider the node with matching
// id as suspected. If the suspect indication, result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Suspect(id int) {
	m.suspected[id] = true
	tl := m.Leader()
	if tl != m.leader {
		for i := 0; i < len(m.subscribers); i++ {
			m.subscribers[i] <- tl
		}
		m.leader = tl
	}
}

// Restore instructs the leader detector to consider the node with matching
// id as restored. If the restore indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Restore(id int) {
	m.suspected[id] = false
	tl := m.Leader()
	if tl >= 0 && tl != m.leader {
		for i := 0; i < len(m.subscribers); i++ {
			m.subscribers[i] <- tl
		}
		m.leader = tl
	}
}

// Subscribe returns a buffered channel which will be used by the leader
// detector to publish the id of the highest ranking node.
// The leader detector will publish UnknownID if all nodes become suspected.
// Subscribe will drop publications to slow subscribers.
// Note: Subscribe returns a unique channel to every subscriber;
// it is not meant to be shared.
func (m *MonLeaderDetector) Subscribe() <-chan int {
	outChannel := make(chan int, 10)
	m.subscribers = append(m.subscribers, outChannel)
	return outChannel
}

func (m *MonLeaderDetector) isSuspected(node int) bool {
	for i, sus := range m.suspected {
		if i == node && sus {
			return true
		}
	}
	return false
}

func (m *MonLeaderDetector) AddNewNode(nodeID int) {
	existed := false
	for _, nID := range m.NodeID {
		if nID == nodeID {
			existed = true
			break
		}
	}
	if existed {
		m.suspected[nodeID] = false
	} else {
		m.NodeID = append(m.NodeID, nodeID)
	}
}

func (m *MonLeaderDetector) ShowSuspected() map[int]bool {
	return m.suspected
}

func (m *MonLeaderDetector) Deactivate() {
	m.alive = false
}
