package bonus

import (
	log "github.com/sirupsen/logrus"
	"net"
)


func GetGatewayStatus() (bool,bool) {
	tcpaddr,err:=net.ResolveTCPAddr("tcp6","computemaster.bxcearth.com:6443")
	if err != nil {
		log.Error(err)
		return false,false
	}
	conn,err:=net.DialTCP("tcp6",nil,tcpaddr)
	if err != nil {
		log.Debug(err)
		return true,false
	}
	log.Debug(conn.RemoteAddr(),conn.LocalAddr())
	defer conn.Close()
	return true,true
}
