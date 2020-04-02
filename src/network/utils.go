package network

import (
	"github.com/qinghon/system/tools"
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

func PatchPpp() error {
	// 添加自动检测通不通的参数 lcp-echo-interval 和lcp-echo-failure
	patchShell:=`
		#!/bin/bash
		cd /etc/ppp/peers
		config=(
		    "lcp-echo-interval 30"
		    "lcp-echo-failure 2"
		)
		for f in $(ls); do
		    for c in "${!config[@]}"; do
		        echo "${config[$c]}" 
		        if ! grep -q "^$(echo ${config[$c]}|awk '{print $1}')" $f ; then
		            echo "${config[$c]}" >>$f
		        fi
		    done
		    echo "pacthed ppp config $f"
		done
	`

	return tools.RunCommand(patchShell)
}
