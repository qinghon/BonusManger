package network

import (
	"log"
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

