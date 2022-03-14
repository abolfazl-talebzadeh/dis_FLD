package failuredetector

import (
	"reflect"
	"testing"
	"time"
)

// Failure detector tests - DO NOT EDIT

func TestAllNodesShouldBeAlivePreStart(t *testing.T) {
	acc := NewAccumulator()
	hbOut := make(chan Heartbeat, 16)
	fd := NewEvtFailureDetector(ourID, clusterOfThree, acc, delta, hbOut)
	done := setTestingHook(fd)

	fd.Start()
	<-done

	if len(fd.alive) != len(clusterOfThree) {
		t.Errorf("TestAllNodesShouldBeAlivePreStart: alive set contains %d node ids, want %d", len(fd.alive), len(clusterOfThree))
	}

	for _, id := range clusterOfThree {
		alive := fd.alive[id]
		if !alive {
			t.Errorf("TestAllNodesShouldBeAlivePreStart: node %d was not set alive", id)
			continue
		}
	}
}

func TestSendReplyToHeartbeatRequest(t *testing.T) {
	acc := NewAccumulator()
	hbOut := make(chan Heartbeat, 16)
	fd := NewEvtFailureDetector(ourID, clusterOfThree, acc, delta, hbOut)
	done := setTestingHook(fd)

	fd.Start()
	<-done

	for i := 0; i < 10; i++ {
		hbReq := Heartbeat{To: ourID, From: i, Request: true}
		fd.DeliverHeartbeat(hbReq)
		<-done
		select {
		case hbResp := <-hbOut:
			if hbResp.To != i {
				t.Errorf("TestSendReplyToHBRequest: Want heartbeat response to id %d, got id %d", i, hbResp.To)
			}
			if hbResp.From != ourID {
				t.Errorf("TestSendReplyToHBRequest: Want heartbeat response from id %d, got id %d", ourID, hbResp.From)
			}
			if hbResp.Request {
				t.Errorf("TestSendReplyToHBRequest: Want heartbeat of type response, got %v", hbResp)
			}
		default:
			t.Errorf("TestSendReplyToHBRequest: expected heartbeat response from %d", i)
		}
	}
}

func TestSetAliveDueToHeartbeatReply(t *testing.T) {
	acc := NewAccumulator()
	hbOut := make(chan Heartbeat, 16)
	fd := NewEvtFailureDetector(ourID, clusterOfThree, acc, delta, hbOut)
	done := setTestingHook(fd)

	fd.Start()
	<-done

	for i := 0; i < 15; i++ {
		hbReply := Heartbeat{To: ourID, From: i, Request: false}
		fd.DeliverHeartbeat(hbReply)
		<-done
		select {
		case hb := <-hbOut:
			t.Errorf("TestSetAliveDueToHBReply: want no outgoing heartbeat, got %v", hb)
		default:
		}
		alive := fd.alive[hbReply.From]
		if !alive {
			t.Errorf("TestSetAliveDueToHBReply: got heartbeat reply from %d, but node was not marked as alive", i)
		}
	}
}

var timeoutTests = []struct {
	alive             map[int]bool
	suspected         map[int]bool
	wantPostSuspected map[int]bool
	wantSuspects      []int
	wantRestores      []int
	wantDelay         time.Duration
}{
	{
		map[int]bool{2: true, 1: true, 0: true},
		map[int]bool{},
		map[int]bool{},
		[]int{},
		[]int{},
		delta,
	},
	{
		map[int]bool{2: true, 1: true, 0: true},
		map[int]bool{0: true},
		map[int]bool{},
		[]int{},
		[]int{0},
		delta + delta,
	},
	{
		map[int]bool{2: true, 0: true},
		map[int]bool{1: true},
		map[int]bool{1: true},
		[]int{},
		[]int{},
		delta,
	},
	{
		map[int]bool{2: true, 1: true, 0: true},
		map[int]bool{1: true},
		map[int]bool{},
		[]int{},
		[]int{1},
		delta + delta,
	},
	{
		map[int]bool{},
		map[int]bool{},
		map[int]bool{2: true, 1: true, 0: true},
		[]int{2, 1, 0},
		[]int{},
		delta,
	},
	{
		map[int]bool{2: true},
		map[int]bool{},
		map[int]bool{1: true, 0: true},
		[]int{1, 0},
		[]int{},
		delta,
	},
	{
		map[int]bool{2: true, 1: true, 0: true},
		map[int]bool{2: true, 1: true, 0: true},
		map[int]bool{},
		[]int{},
		[]int{2, 1, 0},
		delta + delta,
	},
	{
		map[int]bool{},
		map[int]bool{2: true, 1: true, 0: true},
		map[int]bool{2: true, 1: true, 0: true},
		[]int{},
		[]int{},
		delta,
	},
}

func TestTimeoutProcedure(t *testing.T) {
	for i, test := range timeoutTests {
		acc := NewAccumulator()
		hbOut := make(chan Heartbeat, 16)
		fd := NewEvtFailureDetector(ourID, clusterOfThree, acc, delta, hbOut)
		done := setTestingHook(fd)

		// Wait until blocked
		fd.Start()
		defer fd.Stop()
		<-done

		// Set our test data
		fd.alive = test.alive
		fd.suspected = test.suspected

		// Trigger timeout procedure
		fd.timeout()

		// Alive set should always be empty
		if len(fd.alive) > 0 {
			t.Errorf("TestTimeoutProcedure %d: Alive set should always be empty after timeout procedure completes, has length %d", i, len(fd.alive))
		}

		if !reflect.DeepEqual(test.wantPostSuspected, fd.suspected) {
			t.Errorf("TestTimeoutProcedure %d: suspected set post timeout procedure differs", i)
			printSuspectedDiff(t, fd.suspected, test.wantPostSuspected)
		}

		// Check delay
		if fd.delay != test.wantDelay {
			t.Errorf("TestTimeoutProcedure %d: want %v delay after timeout procedure, got %v", i, test.wantDelay, fd.delay)
		}

		// Check for the suspects we want
		if !setEqualSliceID(acc.Suspects, test.wantSuspects) {
			t.Errorf("TestTimeoutProcedure %d: expected and actual set of suspect indications differ", i)
			printFDIndDiff(t, "suspect", acc.Suspects, test.wantSuspects)
		}

		// Check for the restores we want
		if !setEqualSliceID(acc.Restores, test.wantRestores) {
			t.Errorf("TestTimeoutProcedure %d: expected and actual set of restore indications differ", i)
			printFDIndDiff(t, "restore", acc.Restores, test.wantRestores)
		}

		// Check outgoing heartbeat requests
		var (
			outgoingHBs         []Heartbeat
			expectedOutgoingHBs = createExpectedOutgoingHBSet(clusterOfThree)
		)

	hbReqCollect:
		for {
			select {
			case hbReq := <-hbOut:
				outgoingHBs = append(outgoingHBs, hbReq)
			default:
				break hbReqCollect
			}
		}

		if !setEqualSliceHB(expectedOutgoingHBs, outgoingHBs) {
			t.Errorf("TestTimeoutProcedure %d: expected and actual set of outgoing heartbeat requests differ", i)
			printHBDiff(t, outgoingHBs, expectedOutgoingHBs)
		}
	}
}
