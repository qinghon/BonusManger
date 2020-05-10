package network

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

const FdefaultOptionsFile = "/etc/ppp/options"
const F0002replacedefaultroute = "/etc/ppp/ip-up.d/0002replacedefaultroute"
const F0003checkconnect = "/etc/ppp/ip-up.d/0003checkconnect"
const F0002restoreroute = "/etc/ppp/ip-down.d/0002restoreroute"
const F0003closecheck = "/etc/ppp/ip-down.d/0003closecheck"

const installPppScript = `
#!/bin/sh
if which pppd >/dev/null ; then
    exit 0
fi
if which apk >/dev/null ; then
    apk add ppp-pppoe
elif which apt >/dev/null ; then
    apt update
    apt install -y pppoe
elif which yum ; then
    yum install -y ppp
fi
`
const defaultOptions = `
noauth
crtscts
# lock // disable multiple pppoe connect
hide-password
modem
lcp-echo-interval 30
lcp-echo-failure 4
plugin rp-pppoe.so
maxfail 0
persist
noipdefault
defaultroute
+ipv6
`

const checkconect = `#!/bin/bash

# This script is called with the following arguments:
#    Arg  Name                          Example
#    $1   Interface name                ppp0
#    $2   The tty                       ttyS1
#    $3   The link speed                38400
#    $4   Local IP number               12.34.56.78
#    $5   Peer  IP number               12.34.56.99
#    $6   Optional ipparam'' value    foo

PATH=/usr/local/sbin:/usr/sbin:/sbin:/usr/local/bin:/usr/bin:/bin


PPP_IFACE="$1"
PPP_TTY="$2"
PPP_SPEED="$3"
PPP_LOCAL="$4"
PPP_REMOTE="$5"
PPP_IPPARAM="$6"
# "OR" check
_PING=$_PING #set _PING=223.5.5.5,8.8.8.8
_HTTP=$_HTTP #set _HTTP=baidu.com  

export PPP_IFACE PPP_TTY PPP_SPEED PPP_LOCAL PPP_REMOTE PPP_IPPARAM 

# //todo 1 why kill -1 ppp.pid pppd exite
# //todo 2 why ping second ip fail
(
sleep 25
[[ -f "/var/run/ppp-check-$CALL_FILE.pid" ]] &&exit 0
env >>"/var/run/ppp-check-$CALL_FILE.log"
pingcount=$(echo "$_PING"|awk -F, '{print NF}')
httpcount=$(echo "$_HTTP"|awk -F, '{print NF}')
while :; do
    sleep 30
    echo "$$" > "/var/run/ppp-check-$CALL_FILE.pid"
    echo "pid $$" >> "/var/run/ppp-check-$CALL_FILE.log"
    echo "$_PING $_HTTP" >> "/var/run/ppp-check-$CALL_FILE.log"
    if [[ -z "$_PING" ]]&&[[  -z "$_HTTP" ]]; then
        exit 0
    fi
    if [[ -n "$_PING" ]]; then
        count=0
        for addr in $(echo "$_PING"|awk -F, '{for (i = 0; ++i <= NF;) print $i}'); do
            if ! ping -w 3 -I "$PPP_IFACE" "$addr" >>/var/run/ppp-check-$CALL_FILE.log 2>&1 ; then
                count=$((count+1))
            fi
        done
        if [[ $count -ge $pingcount ]]; then
            kill -1 "$PPPD_PID"
            rm "/var/run/ppp-check-$CALL_FILE.pid"
            exit 1
        fi
    fi
    if [[ -n "$_HTTP" ]]; then
        count=0
        for addr in $(echo "$_HTTP"|awk -F, '{for (i = 0; ++i <= NF;) print $i}'); do
            if ! curl  -fs "$addr" >>/var/run/ppp-check-$CALL_FILE.log 2>&1 ; then
                count=$((count+1))
            fi
        done
        if [[ $count -ge $httpcount ]]; then
            kill -1 "$PPPD_PID" 
            rm "/var/run/ppp-check-$CALL_FILE.pid"
            exit 1
        fi
    fi
done) &
`
const closecheck = `#!/bin/bash

# This script is called with the following arguments:
#    Arg  Name                          Example
#    $1   Interface name                ppp0
#    $2   The tty                       ttyS1
#    $3   The link speed                38400
#    $4   Local IP number               12.34.56.78
#    $5   Peer  IP number               12.34.56.99
#    $6   Optional ''ipparam'' value    foo

PATH=/usr/local/sbin:/usr/sbin:/sbin:/usr/local/bin:/usr/bin:/bin

PPP_IFACE="$1"
PPP_TTY="$2"
PPP_SPEED="$3"
PPP_LOCAL="$4"
PPP_REMOTE="$5"
PPP_IPPARAM="$6"
export PPP_IFACE PPP_TTY PPP_SPEED PPP_LOCAL PPP_REMOTE PPP_IPPARAM
# env >/tmp/downenv.txt


kill -15 "$(cat "/var/run/ppp-check-$CALL_FILE.pid")"
rm -vf "/var/run/ppp-check-$CALL_FILE.pid"

`
const replacedefaultroute = `#!/bin/sh

# set metric=600 

METRIC="$metric"
if [ -z "$METRIC" ]; then
    exit 0
fi
if [ -n "$METRIC" ]; then
    if [ $METRIC -eq 0 ]; then
        ip r |grep default|head -n 1 > "/var/run/ppp-defaultroute"
        ip route del $(ip r |head -n 1)
    fi
    ip route add default via "$5" dev "$1" proto static metric "$METRIC"
fi
`
const restoreroute = `#!/bin/sh


# set metric=600 
METRIC="$metric"
if [ -z "$METRIC" ]; then
    exit 0
fi
if [ -n "$METRIC" ]; then
    [ ! -f  "/var/run/ppp-defaultroute" ] && exit 0
    if [ "$METRIC" -eq 0 ]; then
        ip route add $(cat /var/run/ppp-defaultroute)
    fi
fi
rm -vf /var/run/ppp-defaultroute
`

func SetAllAuto() error {
	var err error
	ppps := ReadDslFile()
	for _, pa := range ppps {
		if pa.Status.Enable {
			continue
		}
		if pa.Username != "" && pa.Password != "" {
			err = pa.SetAutoStart()
			if err != nil {
				return err
			}
			err = pa.Connect()
			if err != nil {
				log.Error(err)
			}
		}
	}
	return err
}

func SetAllScript() error {
	var err error
	if PathExist("/etc/ppp/ip-up.d/0001setmetric") {
		_ = os.Remove("/etc/ppp/ip-up.d/0001setmetric")
	}
	err = setFile(F0002replacedefaultroute, replacedefaultroute)
	err = setFile(F0002restoreroute, restoreroute)
	err = setFile(F0003checkconnect, checkconect)
	err = setFile(F0003closecheck, closecheck)
	err = setFile(FdefaultOptionsFile, defaultOptions)
	return err
}

func setFile(_filepath, context string) error {
	if PathExist(_filepath) {
		return nil
	}
	return ioutil.WriteFile(_filepath, []byte(context), 0755)
}
