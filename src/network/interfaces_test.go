package network

import (
	"encoding/json"
	"log"
	"testing"
)

func TestLoad(t *testing.T) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	Is,err:=Load(`
# interfaces(5) file used by ifup(8) and ifdown(8)
auto lo
iface lo inet loopback

auto br0
iface br0 inet static
address 192.168.1.6
broadcast 192.168.1.255
netmask 255.255.255.0
gateway 192.168.1.1
bridge_ports enp1s0
bridge_stp off
bridge_fd 0
bridge_maxwait 0
bridge_maxage 12


#pre-up iptables-restore < /etc/iptables-rules
`)
	if err != nil {
		t.Error(err)
	}
	js,_:=json.Marshal(Is)
	t.Log(string(js))
}

func TestUnmarshal(t *testing.T) {
	var block [][]byte
	block=append(block,[]byte("auto br0"))
	block=append(block,[]byte("iface br0 inet static"))
	block=append(block,[]byte("address 192.168.1.6"))
	block=append(block,[]byte("broadcast 192.168.1.255"))
	block=append(block,[]byte("netmask 255.255.255.0"))
	block=append(block,[]byte("gateway 192.168.1.1"))
	block=append(block,[]byte("bridge_ports enp1s0"))
	block=append(block,[]byte("bridge_stp off"))
	block=append(block,[]byte("bridge_fd 0"))
	block=append(block,[]byte(""))
	block=append(block,[]byte("bridge_maxwait 0"))
	block=append(block,[]byte("bridge_maxage 12"))
	block=append(block,[]byte(""))
	var net *NetInterface
	net,err:=Unmarshal(block)
	if err != nil {
		t.Error(err)
	}
	js,_:=json.Marshal(net)
	t.Log(string(js))
}
func TestUnmarshal2(t *testing.T) {
	var jssa string
	jssa=`{"address":"192.168.1.6","netmask":"255.255.255.0","gateway":"192.168.1.1","broadcast":"192.168.1.255","mtu":"","dns_domain":"","dns_nameservers":"","pre_up":"","post_down":"","up":"","down":"","provider":"","BridgePorts":"","bridge_stp":"","bridge_fd":"","bridge_maxwait":"","bridge_maxage":"","bridge_waitport":"","wpa_ssid":"","wpa_psk":"","wireless_essid":"","wireless_key1":"","wireless_key2":"","wireless_key3":"","wireless_kefaultkey":"","wireless_keymode":""}`
	var net NetOptions
	err:=json.Unmarshal([]byte(jssa),&net)
	if err == nil {
		t.Log(err)
	}
	t.Log(net)
}
func BenchmarkUnmarshal(b *testing.B) {
	var block [][]byte
	block=append(block,[]byte("auto br0"))
	block=append(block,[]byte("iface br0 inet static"))
	block=append(block,[]byte("address 192.168.1.6"))
	block=append(block,[]byte("broadcast 192.168.1.255"))
	block=append(block,[]byte("netmask 255.255.255.0"))
	block=append(block,[]byte("gateway 192.168.1.1"))
	block=append(block,[]byte("bridge_ports enp1s0"))
	block=append(block,[]byte("bridge_stp off"))
	block=append(block,[]byte("bridge_fd 0"))
	block=append(block,[]byte("bridge_maxwait 0"))
	block=append(block,[]byte("bridge_maxage 12"))
	//var net *NetInterface
	b.N=500000
	for i := 0; i < b.N; i++ {
		_,err:=Unmarshal(block)
		if err!=nil {
			b.Error(err)
			break
		}
	}
}
