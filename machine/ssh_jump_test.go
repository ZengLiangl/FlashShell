package machine

import (
	"testing"

	"FlashDock/define"
)

func TestParseJumpEndpointHostPort(t *testing.T) {
	ep, err := ParseJumpEndpoint("jump.example.com:2222")
	if err != nil {
		t.Fatal(err)
	}
	if ep.Host != "jump.example.com" || ep.Port != 2222 {
		t.Fatalf("unexpected endpoint: %+v", ep)
	}
}

func TestParseJumpEndpointUserHost(t *testing.T) {
	ep, err := ParseJumpEndpoint("admin@bastion:22")
	if err != nil {
		t.Fatal(err)
	}
	if ep.User != "admin" || ep.Host != "bastion" || ep.Port != 22 {
		t.Fatalf("unexpected endpoint: %+v", ep)
	}
}

func TestResolveJumpEndpointMachineName(t *testing.T) {
	SetMachineResolver(func(name string) *define.Machine {
		if name != "bastion" {
			return nil
		}
		m := &define.Machine{Name: "bastion"}
		_ = m.SetSensitiveData(&define.SensitiveData{
			Host: "10.0.0.1",
			Port: 22,
			User: "jumpuser",
		})
		return m
	})
	defer SetMachineResolver(nil)

	ep, jumpMachine, err := ResolveJumpEndpoint("bastion", nil)
	if err != nil {
		t.Fatal(err)
	}
	if jumpMachine == nil || ep.Host != "10.0.0.1" || ep.User != "jumpuser" {
		t.Fatalf("unexpected resolve: machine=%v ep=%+v", jumpMachine, ep)
	}
}
