package network

import (
	"io/ioutil"
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
	pa.Name = "wlp3s0"
	out, err := pa.Check(nil, 1, 7)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(out))
}
func TestResolveDslFile1(t *testing.T) {
	ioutil.WriteFile("/tmp/ResolveDslFile1", []byte(`
noipdefault
defaultroute
replacedefaultroute
hide-password
noauth
persist
plugin rp-pppoe.so
user "admin1"
nic-ztzlgmojsa
mtu 1492 
lcp-echo-interval 30
lcp-echo-failure 2
`), 0644)
	pa, err := ResolveDslFile("/tmp/ResolveDslFile1")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(pa.Conf.Interface)
	t.Log(pa)
}
func TestResolveDslFile2(t *testing.T) {
	ioutil.WriteFile("/tmp/ResolveDslFile2", []byte(`
noipdefault
defaultroute
replacedefaultroute
hide-password
noauth
persist
plugin rp-pppoe.so
user "admin1"
ztzlgmojsa
mtu 1492" 
lcp-echo-interval 30
lcp-echo-failure 2
`), 0644)
	pa, err := ResolveDslFile("/tmp/ResolveDslFile2")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(pa.Conf.Interface)
	t.Log(pa)
}
