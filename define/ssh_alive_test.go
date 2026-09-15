package define

import "testing"

func TestRemoteMachineIsConnectedNil(t *testing.T) {
	var rm *RemoteMachine
	if rm.IsConnected() {
		t.Fatal("nil should be disconnected")
	}
	rm = NewRemoteMachine()
	if rm.IsConnected() {
		t.Fatal("empty should be disconnected")
	}
}
