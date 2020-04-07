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


func TestPppoeAccount_Check(t *testing.T) {
	var pa PppoeAccount
	pa.Name="wlp3s0"
	out,err:=pa.Check(nil,1,7)
	if err!=nil {
		t.Error(err)
	}
	t.Log(string(out))
}