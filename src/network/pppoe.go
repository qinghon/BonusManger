package network

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"system/tools"
)

type PppConf struct {
	Interface string `json:"interface"`
	Mtu       int    `json:"mtu"`
	//Linkname string `json:"linkname"`
	Other []string `json:"other"`
}
type PppStatus struct {
	Pid   int      `json:"pid"`
	Iface string   `json:"iface"`
	IP    []string `json:"ip"`
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

func Setppp(p PppoeAccount) error {

	//log.Println(p.Conf)
	fs, err := os.OpenFile("/etc/ppp/peers/"+p.Name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	conf_str := strings.Join(p.Conf.Other, "\n")
	conf_str += fmt.Sprintf("\nuser \"%s\"", p.Username)
	conf_str += fmt.Sprintf("\nnic-%s", p.Conf.Interface)
	if p.Conf.Mtu != 0 {
		conf_str += fmt.Sprintf("\nmtu %d ", p.Conf.Mtu)
	}
	conf_str += "\n"
	//log.Print(conf_str)
	_, err = fs.WriteString(conf_str)
	if err != nil {
		return err
	}
	if err := SetSecrets(p, "/etc/ppp/chap-secrets"); err != nil {
		return err
	}
	if err := SetSecrets(p, "/etc/ppp/pap-secrets"); err != nil {
		return err
	}
	return setpppAuto(p)
}

func setpppAuto(p PppoeAccount) error {
	inface := fmt.Sprintf(`
auto %s
iface %s inet ppp
pre-up /bin/ip link set %s up  # line maintained by bonusmanger
provider %s
`, p.Name, p.Name, p.Conf.Interface, p.Name)
	by, err := ioutil.ReadFile("/etc/network/interfaces")
	if strings.Contains(string(by), fmt.Sprintf("auto %s", p.Name)) {
		return nil
	}
	fp, err := os.OpenFile("/etc/network/interfaces", os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()
	_, err = fp.WriteString(inface)
	if err != nil {
		return err
	}
	return nil
}
/*func delppp_auto(name string) {
	// name : dsl file name //todo 删除interfaces文件中的自启动拨号

}*/
func RunPpp(p PppoeAccount) ([]byte, error) {

	cmd := exec.Command("pppd", "call", p.Name)
	//err := cmd.Start()
	//if err != nil {
	//	return nil, err
	//}
	return cmd.Output()
}


func KillPpp(name string) error {
	return tools.RunCommand(fmt.Sprintf("kill -TERM `cat /var/run/ppp-%s.pid|head -n 1`", name))
}
func getPppStatus(p PppoeAccount) PppoeAccount {
	pid_file := fmt.Sprintf("/var/run/ppp-%s.pid", p.Name)
	if !PathExist(pid_file) {
		log.Printf("not found pid file: %s", pid_file)
		p.Status = PppStatus{0, "", nil}
		return p
	}
	content, err := ioutil.ReadFile(pid_file)
	if err != nil {
		log.Printf("read pid file %s fail: %s", pid_file, err)
		p.Status = PppStatus{0, "", nil}
		return p
	}
	spl := strings.Split(string(content), "\n")
	log.Println(spl)
	if len(spl) == 1 && spl[0] != "" {
		i, err := strconv.Atoi(spl[0])
		if err != nil {
			log.Println(err)
			i = 0
		}
		p.Status = PppStatus{i, "", nil}
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
		p.Status = PppStatus{0, "", nil}
		return p
	}
	return p
}
func ReadDslFile() []PppoeAccount {

	var configs []PppoeAccount
	files := GetFilelist("/etc/ppp/peers")
	if len(*files) == 0 {
		return []PppoeAccount{}
	}
	for _, c := range *files {
		tmp, err := ResolveDslFile(c)
		if err != nil {
			log.Printf("resolve file %s fail", err)
			continue
		}

		configs = append(configs, *tmp)
	}
	configs = ReadChapSecrets(configs)
	for i, p := range configs {
		configs[i] = getPppStatus(p)

	}
	//log.Println(configs)
	return configs
}
func ResolveDslFile(f_path string) (*PppoeAccount, error) {
	var p PppoeAccount
	fd, err := os.Open(f_path)
	if err != nil {
		log.Printf("open %s failed,error :%s", f_path, err)
		return nil, err
	}
	p.Name = filepath.Base(f_path)
	defer fd.Close()
	br := bufio.NewReader(fd)
	mtu_reg, err := regexp.Compile("mtu.?(.*)")
	user_reg, err := regexp.Compile("user.?\"(.*)\"")
	for {
		l, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		tmp_s := strings.TrimSpace(string(l))
		if len(tmp_s) == 0 {
			continue
		}
		if []byte(tmp_s)[:1][0] == []byte("#")[0] {
			continue
		}
		if strings.Contains(tmp_s, "nic-") {
			p.Conf.Interface = tmp_s[4:]
			continue
		}
		if strings.Contains(tmp_s, "user ") {
			sub := user_reg.FindSubmatch([]byte(tmp_s))
			if len(sub) <= 1 {
				log.Printf("not found user in %s", f_path)
			} else {
				p.Username = string(sub[1])
			}
			continue
		}
		if strings.Contains(tmp_s, "mtu") {
			sub := mtu_reg.FindSubmatch([]byte(tmp_s))
			if len(sub) <= 1 {
				log.Printf("not found mtu in %s", f_path)
			} else {
				p.Conf.Mtu, err = strconv.Atoi(string(sub[1]))
				if err != nil {
					log.Printf("conver mtu string to int error: %s", err)
				}
			}
			continue
		}
		p.Conf.Other = append(p.Conf.Other, tmp_s)
	}
	return &p, nil
}

func GetFilelist(path string) *[]string {
	files := &[]string{}
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		//println(path)
		*files = append(*files, path)
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
		tmp_s := strings.TrimSpace(string(l))
		if []byte(tmp_s)[:1][0] == []byte("#")[0] {
			continue
		}
		tmp_s = strings.ReplaceAll(tmp_s, "\"", "")
		tmp_s_s := strings.Split(tmp_s, " ")
		if len(tmp_s_s) <= 2 {
			continue
			log.Println("not split passwd line")
		}
		for i, a := range acc {
			if a.Username == tmp_s_s[0] {
				acc[i].Password = tmp_s_s[2]
			}
		}
		//tmp_p := PppoeAccount{}
		//tmp_p.Username = tmp_s_s[0]
		//tmp_p.Password = tmp_s_s[2]
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

func GetNetworkCard() []networkCard {
	netIs, err := net.Interfaces()
	if err != nil {
		log.Printf("fail to get net interfaces: %v", err)
		return nil
	}

	net_cards := []networkCard{}
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
		net_cards = append(net_cards, tmp)
	}
	return net_cards
}

func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
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
