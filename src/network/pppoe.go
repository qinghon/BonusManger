package network

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type PppConf struct {
	Interface string `json:"interface"`
	Mtu       int    `json:"mtu"`
	//Linkname string `json:"linkname"`
	Other []string `json:"other"`
}
type PppStatus struct {
	Pid    int      `json:"pid"`
	Iface  string   `json:"iface"`
	IP     []string `json:"ip"`
	Enable bool     `json:"enable"`
	Check  bool     `json:"-"`
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

// Save to config file
type CheckConf struct {
	Interval int      `yaml:"interval"`
	Address  []string `yaml:"address"`
	Type     uint8    `yaml:"type"`
}
type StatusStack struct {
	Cmd *exec.Cmd
	PA  PppoeAccount
}

var PPP_POOL map[string]*StatusStack
var CMD_CHAN chan *exec.Cmd

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

const CheckPPPInterval = 10

func Setppp(p PppoeAccount) error {

	//log.Println(p.Conf)
	fs, err := os.OpenFile("/etc/ppp/peers/"+p.Name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer fs.Close()
	confStr := strings.Join(p.Conf.Other, "\n")
	confStr += fmt.Sprintf("\nuser \"%s\"", p.Username)
	confStr += fmt.Sprintf("\n%s", p.Conf.Interface)
	if p.Conf.Mtu != 0 {
		confStr += fmt.Sprintf("\nmtu %d ", p.Conf.Mtu)
	}
	confStr += "\n"
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
	return nil
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
/*func RunPpp(p PppoeAccount) ([]byte, error) {

	cmd := exec.Command("pppd", "nodetach", p.Name)
	//err := cmd.Start()
	//if err != nil {
	//	return nil, err
	//}
	return cmd.Output()
}*/

/*func KillPpp(name string) error {
	return tools.RunCommand(fmt.Sprintf("kill -TERM `cat /var/run/ppp-%s.pid|head -n 1`", name))
}*/
/*
func getPppStatus(p PppoeAccount) PppoeAccount {
	pid_file := fmt.Sprintf("/var/run/ppp-%s.pid", p.Name)
	if !PathExist(pid_file) {
		log.Warn("not found pid file: %s", pid_file)

		p.Status = PppStatus{0, "", nil, true}
		return p
	}
	content, err := ioutil.ReadFile(pid_file)
	if err != nil {
		log.Error("read pid file %s fail: %s", pid_file, err)
		p.Status = PppStatus{0, "", nil, true}
		return p
	}
	spl := strings.Split(string(content), "\n")
	log.Debug(spl)
	if len(spl) == 1 && spl[0] != "" {
		i, err := strconv.Atoi(spl[0])
		if err != nil {
			log.Println(err)
			i = 0
		}
		p.Status = PppStatus{i, "", nil, true}
		return p
	} else if len(spl) > 1 {
		i, err := strconv.Atoi(spl[0])
		if err != nil {
			i = 0
		}
		p.Status.Pid = i
		p.Status.Iface = spl[1]
		iface, err := net.InterfaceByName(spl[1])
		if err != nil {
			log.Println(err)
			p.Status.IP = nil
			return p
		}
		adds, err := iface.Addrs()
		if err != nil {
			log.Println(err)
			p.Status.IP = nil
			return p
		}
		for _, a := range adds {
			p.Status.IP = append(p.Status.IP, a.String())
			return p
		}
	} else {
		p.Status = PppStatus{0, "", nil, true}
		return p
	}
	return p
}
*/

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
func ResolveDslFile(fPath string) (*PppoeAccount, error) {
	var p PppoeAccount
	fd, err := os.Open(fPath)
	if err != nil {
		log.Printf("open %s failed,error :%s", fPath, err)
		return nil, err
	}
	p.Name = filepath.Base(fPath)
	defer fd.Close()
	br := bufio.NewReader(fd)
	mtuReg, err := regexp.Compile("mtu.?(.*)")
	userReg, err := regexp.Compile("user.?\"(.*)\"")
	netCards := GetNetsSampleName()
	nicNetCards := []string{}
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
			p.Conf.Interface = tmpS
			log.Debugf("find interface set as %s", tmpS)
			continue
		}
		if strInArray(nicNetCards, tmpS) != -1 {
			p.Conf.Interface = tmpS
			log.Debugf("find interface set as %s", tmpS)
			continue
		}
		if strings.Contains(tmpS, "user ") {
			sub := userReg.FindSubmatch([]byte(tmpS))
			if len(sub) <= 1 {
				log.Warnf("not found user in %s", fPath)
			} else {
				p.Username = string(sub[1])
			}
			continue
		}
		if strings.Contains(tmpS, "mtu") {
			sub := mtuReg.FindSubmatch([]byte(tmpS))
			if len(sub) <= 1 {
				log.Debugf("not found mtu in %s", fPath)
			} else {
				p.Conf.Mtu, err = strconv.Atoi(string(sub[1]))
				if err != nil {
					log.Debugf("conver mtu string to int error: %s", err)
					p.Conf.Mtu = 1492
				}
			}
			continue
		}
		if strings.Contains(tmpS, " ") {
			sub := strings.Split(tmpS, " ")
			p.Conf.Other = append(p.Conf.Other, sub...)
			continue
		}
		p.Conf.Other = append(p.Conf.Other, tmpS)
	}
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
			if strings.Contains(line, p.Username) {
				lines[i] = fmt.Sprintf("\"%s\" * \"%s\" ", p.Username, p.Password)
			}
		}
		fstr = []byte(strings.Join(lines, "\n"))
		log.Println("found ", p.Username)
	} else {
		log.Println("not found ", p.Username)
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

/*
func CheckLinkAll() error {
	pas := ReadDslFile()
	for _, pa := range pas {
		log.Infof("Start pppoe check progress:%s", pa.Name)
		go pa.GoCheck()

	}
	return nil
}
*/

func (pa *PppoeAccount) RestartPPP() error {
	log.Warnf("Reconnect ppp %s", pa.Name)
	if PPP_POOL[pa.Name] == nil {
		pa.Connect()
		return nil
	}
	cmd := PPP_POOL[pa.Name].Cmd

	if cmd == nil {
		pa.Connect()
		return nil
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		log.Debugf("pppd call %s exited,try to start", pa.Name)
		PPP_POOL[pa.Name].Cmd = nil
		pa.Connect()
		return nil
	}
	return cmd.Process.Signal(syscall.SIGHUP)
}

func (pa *PppoeAccount) Connect() error {
	if PPP_POOL[pa.Name] == nil {
		goto connect
	}
	if PPP_POOL[pa.Name].Cmd != nil {
		log.Warnf("%s connected", pa.Name)
		return nil
	}
	pa.Status.Check = false
connect:
	arg := append([]string{"nodetach"}, pa.Conf.Other...)
	arg = append(arg, pa.Conf.Interface,
		"user", pa.Username,
		"password", pa.Password,
		"logfile", fmt.Sprintf("/var/run/ppp-%s.log", pa.Name),
		"ifname", pa.Name)

	log.Info(strings.Join(arg, "\" \""))
	cmd := exec.Command("pppd", arg...)

	//err := cmd.Start()
	//if err != nil {
	//	return err
	//}
	PPP_POOL[pa.Name] = &StatusStack{cmd, *pa}
	CMD_CHAN <- cmd
	pa.Status.Check = true
	log.Println(fmt.Sprintf("%s starting", pa.Name))
	go pa.GoCheck()
	return nil
}
func (pa *PppoeAccount) Close() error {
	pa.Status.Check = false
	cmd := PPP_POOL[pa.Name].Cmd
	log.Println(pa.Name, cmd)
	if cmd == nil {
		return nil
	}
	if cmd.Process != nil {
		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Error(err)
			PPP_POOL[pa.Name].Cmd = nil
			return err
		}
	}
	PPP_POOL[pa.Name].Cmd = nil
	return nil
}

func (pa *PppoeAccount) Remove() error {
	return os.Remove(path.Join("/etc/ppp/peers", pa.Name))
}

func (pa *PppoeAccount) GoCheck() {
	time.Sleep(time.Second * 15)
	tk := time.NewTicker(time.Second * CheckPPPInterval)
	pa.Status.Check = true
	reCheck := make(chan bool, 2)
	//log.Debug(reCheck)
	for {
		select {
		case <-tk.C:
			_, err := pa.Check(nil, 2, 7)
			if !pa.Status.Check {
				return
			}
			if err != nil {
				log.Debug(err)
				reCheck <- false
			}
		case <-reCheck:
			if !pa.Status.Check {
				return
			}
			_, err := pa.Check(nil, 2, 7)
			if err != nil {
				log.Debug(pa.Status.Check)
				go pa.RestartPPP()
			}
		case <-time.After(time.Second * CheckPPPInterval * 2):
			log.Debug("Check pppoe timeout")
		}
		if pa.Status.Check == false {
			return
		}
		log.Debug("for loop for this!")
	}
	return
}

/*
t: type: 一个字节
value: 11100000 http+ping+ping网关
                 1    1    1      00000
*/
func (pa *PppoeAccount) Check(address []string, num int, t uint8) ([]byte, error) {

	log.Debugf("Status Check Type: %d", t)

	if address == nil || len(address) == 0 {
		address = []string{"8.8.8.8", "223.5.5.5"}
		log.Debugf("Address: %s", address)
	}

	var output []byte
	var errNum int
	for _, addres := range address {
		cmd := exec.Command("ping", addres, "-I", pa.Name, "-w", strconv.Itoa(num))
		log.Debug("ping", addres, "-I", pa.Name, "-w", strconv.Itoa(num))
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
	//enable status
	ok, err := regexp.MatchString(".+?bak", pa.Name)
	if !ok || err != nil {
		pa.Status.Enable = false
	}
	if PPP_POOL[pa.Name] == nil {
		return
	}
	cmd := PPP_POOL[pa.Name].Cmd
	if cmd == nil {
		pa.Status.Pid = 0
		return
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		pa.Status.Pid = 0
	}
	log.Debug(cmd)
	pa.Status.Pid = cmd.Process.Pid
	netI, err := net.InterfaceByName(pa.Name)
	if err != nil {
		pa.Status.Iface = ""
	} else {
		pa.Status.Iface = netI.Name
	}
	//todo: 获取进程状态
}
func (pa *PppoeAccount) GetLog() ([]byte, error) {
	if !PathExist(fmt.Sprintf("/var/run/ppp-%s.log", pa.Name)) {
		return nil, os.ErrNotExist
	}
	return ioutil.ReadFile(fmt.Sprintf("/var/run/ppp-%s.log", pa.Name))
}
func (pa *PppoeAccount) CleanLog() {
	_ = ioutil.WriteFile(fmt.Sprintf("/var/run/ppp-%s.log", pa.Name), nil, 0644)
}
func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
