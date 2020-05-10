package network

import (
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

/*获取内网默认路由IP*/
func Getip() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	defer conn.Close()
	log.Println(strings.Split(conn.LocalAddr().String(), ":")[0])
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}

func IpIsPrivate(ip string) bool {
	ipt := net.ParseIP(ip)
	var isPrivate bool
	netCards := GetNetworkCard()
	if len(netCards) == 0 {
		isPrivate = false
	}
	for _, c := range netCards {
		for _, n := range c.Ip {
			_, tmp, err := net.ParseCIDR(n)
			if err != nil {
				continue
			}
			if tmp.Contains(ipt) {
				isPrivate = true
				break
			}
		}
	}
	return isPrivate
}
