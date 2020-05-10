package network

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"reflect"
	"strings"
)

var NetType = []string{"auto", "allow-auto", "allow-hotplug"}

type NetInterfaceHead struct {
	LinkUp        string `nese:"link_up" json:"link_up"`               //auto allow-auto allow-hotplug
	InterfaceName string `nese:"interface_name" json:"interface_name"` //eth0
	Type          string `nese:"type" json:"type"`                     // dhcp static
	Protocol      string `nese:"protocol" json:"protocol"`             //ipv4:inet or v6:inet6
}
type NetInterface struct {
	NetInterfaceHead
	NetOptions
}
type NetOptions map[string][]string

func (n *NetInterface) Save() error {
	temp := `
%s %s
iface %s %s %s\n`
	netHead := fmt.Sprintf(temp, n.LinkUp, n.InterfaceName, n.InterfaceName, n.Protocol, n.Type)
	var netOptions string

	t := reflect.TypeOf(*n)
	v := reflect.ValueOf(*n)
	for i := 0; i < v.NumField(); i++ {
		typefield := t.Field(i)
		valfeild := v.Field(i)
		tagVal := valfeild.String()
		if tagVal != "" {
			tagName := typefield.Tag.Get("nese")
			netOptions += fmt.Sprintf("    %s %s\n", tagName, tagVal)
		}
	}
	netConf := netHead + netOptions
	log.Println(netConf)
	return nil
}

func Load(_path string) ([]NetInterface, error) {
	by, err := ioutil.ReadFile(_path)
	if err != nil {
		return nil, err
	}
	bySplitLines := bytes.Split(by, []byte("\n"))

	for i, bySplitLine := range bySplitLines {
		bySplitLines[i] = bytes.TrimSpace(bySplitLine)
	}
	for i, bySplitLine := range bySplitLines {
		if len(bySplitLine) == 0 {
			continue
		}
		if bySplitLine[0] == []byte("#")[0] {
			bySplitLines[i] = []byte("")
			continue
		}
		if iq := bytes.Index(bySplitLine, []byte("#")); iq != -1 {
			bySplitLines[i] = bySplitLine[iq:]
			bySplitLines[i] = bytes.TrimSpace(bySplitLines[i])
		}
	}

	var blockBegin []int
	for i, bySplitLine := range bySplitLines {
		if len(bySplitLine) == 0 {
			continue
		}
		kv := keyValSplit(bySplitLine)
		//log.Println(kv,string(kv[0]),strInArray(NetType,string(kv[0])))
		if len(kv) == 2 && strInArray(NetType, string(kv[0])) != -1 {
			//log.Println(string(bySplitLine))
			blockBegin = append(blockBegin, i)
			continue
		}
	}
	var blocks [][][]byte

	for i := 0; i < len(blockBegin); i++ {
		if i+1 == len(blockBegin) {
			blocks = append(blocks, bySplitLines[blockBegin[i]:])
			break
		}
		blocks = append(blocks, bySplitLines[blockBegin[i]:blockBegin[i+1]])
	}
	var nets []NetInterface
	for _, block := range blocks {

		netTmp, err := Unmarshal(block)
		if err != nil {
			log.Println(err)
			continue
		}
		nets = append(nets, *netTmp)
	}

	return nets, nil
}

func Unmarshal(block0 [][]byte) (*NetInterface, error) {
	var block [][]byte
	for _, v := range block0 {
		if len(v) > 1 {
			block = append(block, v)
		}
	}
	//log.Println(string(bytes.Join(block,[]byte("\n"))))
	var head *NetInterfaceHead
	var err error

	head, err = UnmarshalHead(block[0:2])

	if err != nil {
		return nil, err
	}
	options := UnmarshalOptions(block[2:])
	return &NetInterface{*head, *options}, nil
}
func UnmarshalHead(block [][]byte) (*NetInterfaceHead, error) {
	if len(block) != 2 {
		return nil, errors.New("Vaild head len. ")
	}
	var head NetInterfaceHead
	line1 := keyValSplit(block[0])
	if len(line1) < 2 {
		return nil, errors.New("Vaild head split. ")
	}
	//log.Println(line1)
	head.LinkUp = string(line1[0])
	head.InterfaceName = string(line1[1])

	line2 := keyValSplit(block[1])

	if len(line2) != 4 {
		log.Println(line2)
		return nil, errors.New("Vaild head iface. ")
	}
	head.Protocol = string(line2[2])
	head.Type = string(line2[3])
	return &head, nil
}
func UnmarshalOptions(block [][]byte) *NetOptions {
	options := make(NetOptions)
	for _, value := range block {
		if len(value) == 0 {
			continue
		}
		ks := keyValSplit(value)
		if len(ks) < 2 {
			continue
		}
		var tmp []string
		for _, v := range ks[1:] {
			tmp = append(tmp, string(v))
		}
		options[string(ks[0])] = tmp
	}
	return &options
}

func GetNetworkCard() []networkCard {
	netIs, err := net.Interfaces()
	if err != nil {
		log.Printf("fail to get net interfaces: %v", err)
		return nil
	}

	var netCards []networkCard
	for _, netI := range netIs {
		tmp := networkCard{}
		if len(netI.HardwareAddr.String()) == 0 {
			continue
		}
		tmp.Macaddr = netI.HardwareAddr.String()
		if strings.Contains(netI.Name, "veth") {
			continue
		} else if strings.Contains(netI.Name, "docker") {
			continue
		} else if strings.Contains(netI.Name, "br-") {
			continue
		}
		tmp.Name = netI.Name
		byName, err := net.InterfaceByName(netI.Name)
		if err != nil {
			log.Println("get interface ", tmp.Name, " failed")
		}
		address, err := byName.Addrs()
		for _, v := range address {
			tmp.Ip = append(tmp.Ip, v.String())
		}
		netCards = append(netCards, tmp)
	}
	return netCards
}
func GetNetsSampleName() []string {
	netCards := GetNetworkCard()
	var netsName []string
	for _, netCard := range netCards {
		netsName = append(netsName, netCard.Name)
	}
	return netsName
}
func GetIFbyName(name string) []net.Addr {
	netI, err := net.InterfaceByName(name)
	if err != nil {
		return nil
	}
	ips, err := netI.Addrs()
	if err != nil {
		return nil
	}
	return ips

}

/*
func Unmarshal(block [][]byte,v interface{}) error {
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)
	log.Println(block)
	for _,value:=range block[2:] {
		if len(value)==0 {
			continue
		}
		ks:=keyValSplit(value)
		if len(ks) != 2 {
			continue
		}
		for i := 0; i < rt.Elem().NumField(); i++ {
			typefield := rt.Elem().Field(i)
			tagName := typefield.Tag.Get("nese")
			log.Println(typefield.Type)
			if tagName==string(ks[0]) {
				newValue:=rv.Elem().Field(i)
				log.Println(tagName,newValue.Type())
				newValue.SetString(string(ks[1]))
			}
		}
	}
	return nil
}*/

func keyValSplit(by []byte) [][]byte {
	var kv [][]byte
	for _, v := range bytes.Split(by, []byte(" ")) {
		if len(v) != 0 {
			kv = append(kv, v)
		}
	}
	return kv
}
func valInArray(ay [][]byte, val []byte) int {
	for i := 0; i < len(ay); i++ {
		if bytes.Equal(ay[i], val) {
			return i
		}
	}
	return -1
}
func strInArray(ay []string, val string) int {
	for i := 0; i < len(ay); i++ {
		if val == ay[i] {
			return i
		}
	}
	return -1
}
