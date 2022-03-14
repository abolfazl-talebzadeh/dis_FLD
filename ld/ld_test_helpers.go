// Test helper functions - DO NOT EDIT

package leaderdetector

type event struct {
	suspect bool // true -> suspect, false -> restore
	id      int  // node id to suspect or restore
}

type eventPubSub struct {
	suspect       bool // true -> suspect, false -> restore
	id            int  // node id to suspect or restore
	produceOutput bool // should the event produce output to subscribers
	wantLeader    int  // what leader id should publication contain
}

var pubSubTestSeq = []eventPubSub{
	{true, 2, true, 1},         // suspect 2, want pub for 1
	{false, 2, true, 2},        // restore 2, want pub for 2
	{false, 2, false, -1},      // restore 2, no change -> no output
	{true, 1, false, -1},       // suspect 1, no change -> no output
	{true, 0, false, -1},       // suspect 0, no change -> no output
	{true, 2, true, UnknownID}, // suspect 2, all suspected, want pub for UnknownID
	{false, 0, true, 0},        // restore 0, want pub for 0
	{false, 1, true, 1},        // restore 1, want pub for 1
	{false, 2, true, 2},        // restore 2, want pub for 2
}
