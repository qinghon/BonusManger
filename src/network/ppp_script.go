package network

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

const DefaultOptionsFile = "/etc/ppp/options"

const setMetric = `#!/bin/sh
# set metric=600 
METRIC="$metric"
if [ -n "$METRIC" ]; then
    ip route add default via "$5" dev "$1" proto static metric "$METRIC"
fi
`
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
const DefaultOptions = `
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

func SetDefaultOptions() error {
	err := ioutil.WriteFile(DefaultOptionsFile, []byte(DefaultOptions), 0644)
	return err
}
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

func SetMetricScript() error {
	if PathExist("/etc/ppp/ip-up.d/0001setmetric") {
		return nil
	}
	err := ioutil.WriteFile("/etc/ppp/ip-up.d/0001setmetric", []byte(setMetric), 0755)
	return err
}
