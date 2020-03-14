package main

import (
	"encoding/json"
	"net"
	"github.com/qinghon/network"
	"testing"
)

func Test_read_chap_secrets(t *testing.T) {
	ret := network.ReadChapSecrets([]network.PppoeAccount{})
	t.Log(ret)
}
func Test_getFilelist(t *testing.T) {
	tmp := network.GetFilelist("/etc/ppp/peers")
	t.Log(*tmp)
}
func Test_Read_dsl_file(t *testing.T) {
	tmp := network.ReadDslFile()
	js, _ := json.Marshal(tmp)
	t.Log(string(js))
	//t.Log(tmp)
}
func Test_get_network_card(t *testing.T) {
	nets := network.GetNetworkCard()
	t.Log(nets)
}
func Test_resolve_dsl_file(t *testing.T) {
	tmp, err := network.ResolveDslFile("/etc/ppp/peers/dsl-provider")
	if err != nil {
		t.Error(err)
	}
	js, err := json.Marshal(tmp)
	t.Log(string(js))
}
func Test_gettime(t *testing.T) {
	iface, err := net.InterfaceByName("docker0")
	if err != nil {
		t.Error(err)
	}
	t.Log(iface.Addrs())
}
