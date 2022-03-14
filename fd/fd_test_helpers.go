// Test helper functions - DO NOT EDIT

package failuredetector

import (
	"testing"
	"time"
)

const (
	ourID = 2
	delta = time.Second
)

var clusterOfThree = []int{2, 1, 0}

func setTestingHook(fd *EvtFailureDetector) <-chan bool {
	done := make(chan bool)
	fd.testingHook = func() {
		fd.timeoutSignal.C = nil
		done <- true
	}
	return done
}

func printSuspectedDiff(t *testing.T, got, want map[int]bool) {
	t.Errorf("-----------------------------------------------------------------------------")
	t.Errorf("Got:")
	if len(got) == 0 {
		t.Errorf("None")
	}
	for id := range got {
		t.Errorf("suspect %d", id)
	}
	t.Errorf("Want:")
	if len(want) == 0 {
		t.Errorf("None")
	}
	for id := range want {
		t.Errorf("suspect %d", id)
	}
	t.Errorf("-----------------------------------------------------------------------------")
}

func printFDIndDiff(t *testing.T, fdIndType string, got map[int]bool, want []int) {
	t.Errorf("-----------------------------------------------------------------------------")
	t.Errorf("Got:")
	if len(got) == 0 {
		t.Errorf("None")
	}
	for id := range got {
		t.Errorf("%s %d", fdIndType, id)
	}
	t.Errorf("Want:")
	if len(want) == 0 {
		t.Errorf("None")
	}
	for _, id := range want {
		t.Errorf("%s %d", fdIndType, id)
	}
	t.Errorf("-----------------------------------------------------------------------------")
}

func setEqualSliceID(a map[int]bool, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for _, val := range b {
		if _, found := a[val]; !found {
			return false
		}
	}
	return true
}

func createHB(to, from int, request bool) Heartbeat {
	return Heartbeat{
		To:      to,
		From:    from,
		Request: request,
	}
}

func createExpectedOutgoingHBSet(requests []int) map[Heartbeat]bool {
	set := make(map[Heartbeat]bool, len(requests))
	for _, req := range requests {
		set[createHB(req, ourID, true)] = true
	}
	return set
}

func setEqualSliceHB(a map[Heartbeat]bool, b []Heartbeat) bool {
	if len(a) != len(b) {
		return false
	}
	for _, val := range b {
		if _, found := a[val]; !found {
			return false
		}
	}
	return true
}

func printHBDiff(t *testing.T, got []Heartbeat, want map[Heartbeat]bool) {
	t.Errorf("-----------------------------------------------------------------------------")
	t.Errorf("Expected and actual set of outgoing heartbeats differ")
	t.Errorf("Got:")
	if len(got) == 0 {
		t.Errorf("None")
	}
	for _, hb := range got {
		t.Errorf("%v", hb)
	}
	t.Errorf("Want:")
	if len(want) == 0 {
		t.Errorf("None")
	}
	for hb := range want {
		t.Errorf("%v", hb)
	}
	t.Errorf("-----------------------------------------------------------------------------")
}

// Accumulator is simply a implementation of the SuspectRestorer interface
// that record the suspect and restore indications it receives. Used for
// testing.
type Accumulator struct {
	Suspects map[int]bool
	Restores map[int]bool
}

func NewAccumulator() *Accumulator {
	return &Accumulator{
		Suspects: make(map[int]bool),
		Restores: make(map[int]bool),
	}
}

func (a *Accumulator) Suspect(id int) {
	a.Suspects[id] = true
}

func (a *Accumulator) Restore(id int) {
	a.Restores[id] = true
}

func (a *Accumulator) Reset() {
	a.Suspects = make(map[int]bool)
	a.Restores = make(map[int]bool)
}
