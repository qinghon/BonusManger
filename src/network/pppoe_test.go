package network

import (
	"testing"
)

func TestGetFilelist(t *testing.T) {
	t.Log(GetFilelist("/etc/ppp/peers"))
}
func TestGetFilelist2(t *testing.T) {
	t.Log(GetFilelist("/etc/network/interfaces.d"))
}
func TestReadDslFile(t *testing.T) {
	t.Log(ReadDslFile())
}
func TestCheckLink(t *testing.T) {
	s:=PppoeAccount{}
	s.Status.Iface="eth0"
	err:=CheckLink(s,1)
	if err != nil {
		t.Error(err)
	}
}