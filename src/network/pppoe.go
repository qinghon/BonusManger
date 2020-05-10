package network

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/qinghon/debinterface"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type PppConf struct {
	User            string   `json:"user"`
	Interface       string   `json:"interface"`
	Mtu             int      `json:"mtu"`
	Metric          int      `json:"metric"`
	Linkname        string   `json:"linkname"`
	Ifname          string   `json:"ifname"`
	Defaultroute    bool     `json:"defaultroute"`
	LcpEchoFailure  int      `json:"lcp-echo-failure"`
	LcpEchoInterval int      `json:"lcp-echo-interval"`
	Usepeerdns      bool     `json:"usepeerdns"`
	Nameservers     []string `json:"nameservers"`
	PingAddr        []string `json:"ping_addr"`
	HttpAddr        []string `json:"http_addr"`
	Other           []string `json:"other"`
}
type PppStatus struct {
	Pid    int      `json:"pid"`
	Iface  string   `json:"iface"`
	IP     []string `json:"ip"`
	Enable bool     `json:"enable"`
}
type PppoeAccount struct {
	Name     string    `json:"name"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	Conf     PppConf   `json:"conf"`
	Status   PppStatus `json:"status"`
}

type networkCard struct {
	Name    string   `json:"name"`
	Macaddr string   `json:"macaddr"`
	Ip      []string `json:"ip"`
}
type NetInterfaces struct {
	Context     string        `json:"context"`
	NetworkCard []networkCard `json:"networkCard"`
}
type chapSecret struct {
	Client    string
	Server    string `default:"*"`
	Secret    string
	IpAddress string
}
type papSecret struct {
	Client   string
	Server   string `default:"*"`
	Password string
	Option   string
}

func (pppconf *PppConf) Export(name, user string) string {
	var conf string
	if user != "" {
		conf += fmt.Sprintf("user \"%s\"\n", user)
	} else if pppconf.User != "" {
		conf += fmt.Sprintf("user \"%s\"\n", pppconf.User)
	}
	if pppconf.Mtu != 0 {
		conf += fmt.Sprintf("mtu %d\n", pppconf.Mtu)
	}
	if pppconf.Linkname != "" {
		conf += fmt.Sprintf("linkname %s\n", pppconf.Linkname)
	} else {
		conf += fmt.Sprintf("linkname %s\n", name)
	}
	if pppconf.Ifname != "" {
		conf += fmt.Sprintf("ifname %s\n", name)
	}
	if pppconf.Defaultroute {
		conf += fmt.Sprintf("set metric=%d\n", pppconf.Metric)
	}
	if pppconf.HttpAddr != nil || len(pppconf.HttpAddr) != 0 {
		conf += fmt.Sprintf("set _HTTP=%s\n", strings.Join(pppconf.HttpAddr, ","))
	}
	if pppconf.PingAddr != nil || len(pppconf.PingAddr) != 0 {
		conf += fmt.Sprintf("set _PING=%s\n", strings.Join(pppconf.HttpAddr, ","))
	}
	conf += fmt.Sprintf("lcp-echo-failure %d\n", pppconf.LcpEchoFailure)
	conf += fmt.Sprintf("lcp-echo-interval %d\n", pppconf.LcpEchoInterval)
	if pppconf.Usepeerdns {
		conf += "usepeerdns\n"
	}
	if pppconf.Interface != "" {
		conf += fmt.Sprintf("%s\n", pppconf.Interface)
	}
	if len(pppconf.Other) != 0 {
		conf += strings.Join(pppconf.Other, "\n")
	}
	conf += fmt.Sprintf("\nlogfile /var/run/ppp-%s.log\n", name)
	return conf
}
func (pppconf *PppConf) Parse(fPath string) error {
	fd, err := os.Open(fPath)
	if err != nil {
		log.Printf("open %s failed,error :%s", fPath, err)
		return err
	}

	defer fd.Close()
	br := bufio.NewReader(fd)

	netCards := GetNetsSampleName()
	var nicNetCards []string
	for _, n := range netCards {
		nicNetCards = append(nicNetCards, "nic-"+n)
	}
	for {
		l, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		tmpS := strings.TrimSpace(string(l))
		if len(tmpS) == 0 {
			continue
		}
		if []byte(tmpS)[:1][0] == []byte("#")[0] {
			continue
		}
		if strInArray(netCards, tmpS) != -1 {
			pppconf.Interface = tmpS
			log.Debugf("find interface set as %s", tmpS)
			continue
		}
		if strInArray(nicNetCards, tmpS) != -1 {
			pppconf.Interface = tmpS
			log.Debugf("find interface set as %s", tmpS)
			continue
		}
		sline := strings.Fields(tmpS)
		if len(sline) <= 0 {
			continue
		}
		switch sline[0] {
		case "user":
			pppconf.User = strings.ReplaceAll(sline[1], "\"", "")
			//p.Username=p.Conf.User
		case "lcp-echo-interval":
			n, err := strconv.Atoi(sline[1])
			if err != nil {
				pppconf.Other = append(pppconf.Other, tmpS)
			}
			pppconf.LcpEchoInterval = n
		case "lcp-echo-failure":
			n, err := strconv.Atoi(sline[1])
			if err != nil {
				pppconf.Other = append(pppconf.Other, tmpS)
			}
			pppconf.LcpEchoFailure = n
		case "mtu":
			n, err := strconv.Atoi(sline[1])
			if err != nil {
				pppconf.Other = append(pppconf.Other, tmpS)
				continue
			}
			pppconf.Mtu = n
		case "linkname":
			pppconf.Linkname = sline[1]
		case "ifname":
			pppconf.Ifname = sline[1]
		case "set":
			setline := strings.Split(sline[1], "=")
			switch setline[0] {
			case "metric":
				n, err := strconv.Atoi(setline[1])
				if err != nil {
					pppconf.Other = append(pppconf.Other, tmpS)
					continue
				}
				pppconf.Defaultroute = true
				pppconf.Metric = n
			case "_HTTP":
				pppconf.PingAddr = strings.Split(setline[1], ",")
			case "_PING":
				pppconf.HttpAddr = strings.Split(setline[1], ",")
			default:
				pppconf.Other = append(pppconf.Other, tmpS)
			}
		case "persist", "noauth", "hide-password", "noipdefault", "defaultroute",
			"modem", "plugin", "maxfail", "logfile":
			continue
		default:
			pppconf.Other = append(pppconf.Other, tmpS)
		}
	}
	return nil
}

func Setppp(p PppoeAccount) error {

	//log.Println(p.Conf)
	fs, err := os.OpenFile("/etc/ppp/peers/"+p.Name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer fs.Close()
	confStr := p.Conf.Export(p.Name, p.Username)

	//log.Print(confStr)
	_, err = fs.WriteString(confStr)
	if err != nil {
		return err
	}
	if err := SetSecrets(p, "/etc/ppp/chap-secrets"); err != nil {
		return err
	}
	if err := SetSecrets(p, "/etc/ppp/pap-secrets"); err != nil {
		return err
	}
	if err := p.SetAutoStart(); err != nil {
		return err
	}
	return p.RestartPPP()
}

/*
func setpppAuto(p PppoeAccount) error {
	inface := fmt.Sprintf(`

auto %s
iface %s inet ppp
pre-up /bin/ip link set %s up  # line maintained by bonusmanger
provider %s

`, p.Name, p.Name, p.Conf.Interface, p.Name)
	by, err := ioutil.ReadFile("/etc/network/interfaces")
	if err != nil {
		return err
	}
	if strings.Contains(string(by), fmt.Sprintf("auto %s", p.Name)) {
		return nil
	}
	if ! bytes.Contains(by, []byte("source /etc/network/interfaces.d/*")) {
		fp, err := os.OpenFile("/etc/network/interfaces", os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		defer fp.Close()
		_, err = fp.WriteString("source /etc/network/interfaces.d/*")
		if err != nil {
			return err
		}
	}
	fppp, err := os.OpenFile(fmt.Sprintf("/etc/network/interfaces.d/%s", p.Name), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := fppp.WriteString(fmt.Sprintf(inface, p.Name)); err != nil {
		return err
	}
	return nil
}
*/
/*func delppp_auto(name string) {
	// name : dsl file name //todo 删除interfaces文件中的自启动拨号

}*/

func ReadDslFile() []PppoeAccount {

	var configs []PppoeAccount
	files := GetFilelist("/etc/ppp/peers")
	if len(files) == 0 {
		return []PppoeAccount{}
	}
	for _, c := range files {
		tmp, err := ResolveDslFile(c)
		if err != nil {
			log.Printf("resolve file %s fail", err)
			continue
		}

		configs = append(configs, *tmp)
	}

	for i := 0; i < len(configs); i++ {
		configs[i].GetStatus()
	}

	return configs
}
func GetFilelist(path string) []string {
	var files []string
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		//println(path)
		files = append(files, path)
		return nil
	})
	if err != nil {
		log.Printf("filepath.Walk() returned %v\n", err)
	}
	return files
}

func ResolveDslFile(fPath string) (*PppoeAccount, error) {
	var p PppoeAccount
	p.Name = filepath.Base(fPath)
	var pppconf PppConf
	err := pppconf.Parse(fPath)
	if err != nil {
		return nil, err
	}
	log.Debug(pppconf)
	p.Username = pppconf.User
	p.Conf = pppconf
	pd, err := getDslPassword(p.Username)
	if err == nil {
		p.Password = pd
	}
	return &p, nil
}
func getDslPassword(username string) (string, error) {
	chapSecrets, err := ResolveChapSecrets()
	papSecrets, err := ResolvePapSecrets()
	if err != nil {
		log.Println(err)
	}

	for _, pap := range papSecrets {
		if username == pap.Client {
			return pap.Password, nil
		}
	}

	for _, chap := range chapSecrets {
		if username == chap.Client {
			return chap.Secret, nil
		}
	}

	return "", errors.New("password not found")
}
func ResolveChapSecrets() ([]chapSecret, error) {
	fc, err := os.Open("/etc/ppp/chap-secrets")
	if err != nil {
		log.Printf("read chap-secrets fail: %s", err)
		return nil, err
	}
	defer fc.Close()
	var chapSecrets []chapSecret
	br := bufio.NewReader(fc)
	for {
		l, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		// tmpS clean space not head or end
		tmpS := strings.TrimSpace(string(l))
		if []byte(tmpS)[:1][0] == []byte("#")[0] {
			continue
		}

		tmpSSES := strings.Split(tmpS, " ")

		for i := 0; i < len(tmpSSES); i++ { //clean " in string
			tmpSSES[i] = strings.ReplaceAll(tmpSSES[i], "\"", "")
		}
		var secrets []string
		for i := 0; i < len(tmpSSES); i++ { //clean no content string in list
			if tmpSSES[i] != "" {
				secrets = append(secrets, tmpSSES[i])
			}
		}
		if len(secrets) <= 2 {
			continue
		}
		chapSecrets = append(chapSecrets, chapSecret{Client: secrets[0], Server: secrets[1], Secret: secrets[2]})
	}
	return chapSecrets, nil
}

func ResolvePapSecrets() ([]papSecret, error) {
	fc, err := os.Open("/etc/ppp/pap-secrets")
	if err != nil {
		log.Printf("read chap-secrets fail: %s", err)
		return nil, err
	}
	defer fc.Close()
	var papSecrets []papSecret
	br := bufio.NewReader(fc)
	for {
		l, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		// tmpS clean space not head or end
		tmpS := strings.TrimSpace(string(l))
		if []byte(tmpS)[:1][0] == []byte("#")[0] {
			continue
		}

		tmpSSES := strings.Split(tmpS, " ")

		for i := 0; i < len(tmpSSES); i++ { //clean " in string
			tmpSSES[i] = strings.ReplaceAll(tmpSSES[i], "\"", "")
		}
		var secrets []string
		for i := 0; i < len(tmpSSES); i++ { //clean no content string in list
			if tmpSSES[i] != "" {
				secrets = append(secrets, tmpSSES[i])
			}
		}
		if len(secrets) <= 2 {
			continue
		}
		papSecrets = append(papSecrets, papSecret{Client: secrets[0], Server: secrets[1], Password: secrets[2]})
	}
	return papSecrets, nil
}

func ReadChapSecrets(acc []PppoeAccount) []PppoeAccount {
	fc, err := os.Open("/etc/ppp/chap-secrets")
	if err != nil {
		log.Printf("read chap-secrets fail: %s", err)
		return nil
	}
	defer fc.Close()
	//acc:=[]PppoeAccount{}
	br := bufio.NewReader(fc)
	for {
		l, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		tmpS := strings.TrimSpace(string(l))
		if []byte(tmpS)[:1][0] == []byte("#")[0] {
			continue
		}
		tmpS = strings.ReplaceAll(tmpS, "\"", "")
		tmpSSES := strings.Split(tmpS, " ")
		if len(tmpSSES) <= 2 {
			continue
			log.Println("not split passwd line")
		}
		for i, a := range acc {
			if a.Username == tmpSSES[0] {
				acc[i].Password = tmpSSES[2]
			}
		}
		//tmp_p := PppoeAccount{}
		//tmp_p.Username = tmpSSES[0]
		//tmp_p.Password = tmpSSES[2]
		//acc = append(acc, tmp_p)
	}
	return acc
}
func SetSecrets(p PppoeAccount, filename string) error {
	//filename:="/etc/ppp/chap-secrets"
	if !PathExist(filename) {
		os.Create(filename)
	}
	fstr, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println("set passwd fail", err)
		return err
	}
	fc, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		log.Println("set passwd fail", err)
		return err
	}
	defer fc.Close()
	//br:=bufio.NewReader(fc)
	//fstr,err:=ioutil.ReadAll(fc)
	//log.Println(bytes.Contains(fstr,[]byte(p.Username)),[]byte(p.Username),string(fstr))
	if bytes.Contains(fstr, []byte(p.Username)) {
		lines := strings.Split(string(fstr), "\n")
		for i, line := range lines {
			if strings.Contains(line, "\""+p.Username+"\"") {
				lines[i] = fmt.Sprintf("\"%s\" * \"%s\" ", p.Username, p.Password)
			}
		}
		fstr = []byte(strings.Join(lines, "\n"))
		log.Println("found ", p.Username)
	} else {
		log.Info("not found ", p.Username)
		//log.Println(fmt.Sprintf("\n\"%s\" * \"%s\" ",p.Username,p.Password))
		fstr = append(fstr, []byte(fmt.Sprintf("\n\"%s\" * \"%s\"\n", p.Username, p.Password))...)
	}
	//log.Print(string(fstr))
	_, err = fc.Write(fstr)
	return err
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(dstName)
	if err != nil {
		return
	}
	defer dst.Close()

	return io.Copy(dst, src)
}
func CheckPpp() ([]byte, error) {
	_, err := exec.LookPath("pppd")
	if err != nil {
		return IntsallPpp()
	}
	if !PathExist("/dev/ppp") {
		return IntsallPpp()
	}
	return nil, nil
}

func IntsallPpp() ([]byte, error) {
	log.Println("sh", "-c", installPppScript)
	cmd := exec.Command("sh", "-c", installPppScript)
	//if err := RunCommand(installPppScript); err != nil {
	//	log.Printf("Install pppoe software failed")
	//	return err
	//}
	return cmd.Output()
}

func (pa *PppoeAccount) SetAutoStart() error {

	adp := debinterface.NewAdapter(pa.Name, "inet", "ppp")
	adp.SetAuto(true)
	adp.SetProvider(pa.Name)

	if len(pa.Conf.Nameservers) != 0 {
		var ipList []net.IP
		for i := 0; i < len(pa.Conf.Nameservers); i++ {
			ipList = append(ipList, net.ParseIP(pa.Conf.Nameservers[i]))
		}
		adp.SetDnsNameServer(ipList)
	}
	var faces debinterface.Interfaces
	faces.FilePath = "/etc/network/interfaces"
	err := faces.Add(adp)
	if err != nil {
		return err
	}
	return nil
}
func (pa *PppoeAccount) UnSetAuto() error {
	adp := debinterface.Interface{}
	adp.SetName(pa.Name)
	//adp.SetAddrFam("inet")
	//adp.SetAuto(true)
	//adp.SetAddrSource("ppp")
	//adp.SetProvider(pa.Name)

	faces := debinterface.Interfaces{}
	faces.FilePath = "/etc/network/interfaces"

	if err := faces.Del(adp); err != nil {
		return err
	}
	return nil
}

func (pa *PppoeAccount) RestartPPP() error {
	log.Warnf("Reconnect ppp %s", pa.Name)
	err := pa.Close()
	if err != nil {
		log.Error(err)
	}
	time.Sleep(time.Second * 3)
	err = pa.Connect()
	return err
}

func (pa *PppoeAccount) Connect() error {

	by, err := IfUp(pa.Name)
	if err != nil {
		return err
		log.Error(string(by))
	}
	log.Println(fmt.Sprintf("%s starting", pa.Name))
	return nil
}
func (pa *PppoeAccount) Close() error {
	log.Warnf("Close: %s", pa.Name)
	by, err := IfDown(pa.Name)
	if err != nil {
		log.Error(string(by))
	}
	return err
}

func (pa *PppoeAccount) Remove() error {

	if err := pa.Close(); err != nil {
		log.Error(err)
	}
	if err := pa.UnSetAuto(); err != nil {
		log.Error(err)
	}

	return os.Remove(path.Join("/etc/ppp/peers", pa.Name))
}

/*
t: type: 一个字节
value: 11100000 http+ping+ping网关
                 1    1    1      00000
*/
func (pa *PppoeAccount) Check(address []string, num int, t uint8) ([]byte, error) {

	log.Debugf("Status Check Type: %d ,%s", t, pa.Name)

	if pa.Status.Iface == "" {
		pa.GetStatus()
		return nil, errors.New("iface not up")
	}
	intf := pa.Status.Iface
	if address == nil || len(address) == 0 {
		address = []string{"8.8.8.8", "223.5.5.5"}
		log.Debugf("Address: %s", address)
	}

	var output []byte
	var errNum int
	for _, addres := range address {
		cmd := exec.Command("ping", addres, "-I", intf, "-w", strconv.Itoa(num))
		log.Debug("ping", " ", addres, " ", "-I", " ", intf, " ", "-w", " ", strconv.Itoa(num), " # ", pa.Name)
		tmpOutput, err := cmd.Output()
		if err != nil {
			log.Debug(string(tmpOutput), err)
			errNum += 1
			output = append(output, tmpOutput...)
		}
	}
	log.Debug(errNum)
	if len(address) <= 2 && errNum >= 1 {
		return output, errors.New("Connect not work. ")
	} else if len(address) > 2 && float64(errNum) > float64(len(address))*0.75 {

		return output, errors.New("Connect not work. ")
	} else {
		log.Debugf("Connect working: %s", pa.Name)
		return output, nil
	}
}
func (pa *PppoeAccount) GetStatus() {
	netI, err := net.InterfaceByName(pa.Name)
	if err != nil {
		pa.Status.Iface = ""
		log.Debugf("not found ifname interface: %s", pa.Name)

	} else {
		pa.Status.Iface = netI.Name
		ips, err := netI.Addrs()
		if err != nil {
			goto getenable
		}
		for _, ip := range ips {
			pa.Status.IP = append(pa.Status.IP, ip.String())
		}
	}
	//todo: 获取进程状态

	if pa.Conf.Linkname != "" {
		log.Debug(pa.Conf.Linkname)
		pidFile := fmt.Sprintf("/var/run/ppp-%s.pid", pa.Name)
		if PathExist(pidFile) {
			by, err := ioutil.ReadFile(pidFile)
			if err != nil {
				log.Error(err)
				goto getenable
			}
			sline := strings.Split(string(by), "\n")
			switch len(sline) {
			case 1:
				s2ip, err := strconv.Atoi(strings.TrimSpace(sline[0]))
				if err == nil {
					pa.Status.Pid = s2ip
				}
			case 2, 3:
				s2ip, err := strconv.Atoi(strings.TrimSpace(sline[0]))
				if err == nil {
					pa.Status.Pid = s2ip
				}
				pa.Status.Iface = strings.TrimSpace(sline[1])
			}
			log.Debug(sline, len(sline), pa.Status)
		} else {
			log.Warnf("%s file not found.", pidFile)
		}
	}
	//pa.Status.Pid = cmd.Process.Pid
	//log.Debug(cmd)

	//enable status
getenable:
	reader := debinterface.NewReader(debinterface.INTERFACES_FILE)
	// todo filepath.Glob bug
	if err := reader.Read(); err != nil {
		log.Error(err)
		return
	}
	for _, i := range reader.Adapters {
		if i.GetName() == pa.Name && pa.Username != "" && pa.Password != "" {
			pa.Status.Enable = true
		}
	}
	log.Debugf("get %s enable status. %t", pa.Name, pa.Status.Enable)
}
func (pa *PppoeAccount) GetLog() ([]byte, error) {
	if !PathExist(fmt.Sprintf("/var/run/ppp-%s.log", pa.Name)) {
		return nil, os.ErrNotExist
	}
	filename := fmt.Sprintf("/var/run/ppp-%s.log", pa.Name)
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if fi.Size() > 1024*512 {
		f.Seek(-1024*512, 2)
	}
	by, err := ioutil.ReadAll(f)
	return by, err
}
func (pa *PppoeAccount) CleanLog() {
	_ = ioutil.WriteFile(fmt.Sprintf("/var/run/ppp-%s.log", pa.Name), nil, 0644)
}

func IfUp(name string) ([]byte, error) {
	cmd := exec.Command("ifup", name, "-v")
	return cmd.Output()
}
func IfDown(name string) ([]byte, error) {
	cmd := exec.Command("ifdown", name, "-v")
	return cmd.Output()
}

func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
