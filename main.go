package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"hardware"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ppp_conf struct {
	Interface string   `json:"interface"`
	Mtu       int      `json:"mtu"`
	//Linkname string `json:"linkname"`
	Other     []string `json:"other"`
}
type Pppoe_account struct {
	Name     string   `json:"name"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Conf     ppp_conf `json:"conf"`
	Status   bool `json:"status"`
}
type network_card struct {
	Name    string   `json:"name"`
	Macaddr string   `json:"macaddr"`
	Ip      []string `json:"ip"`
}
type Message struct {
	Code    int    `json:"code"`
	Details string `json:"details"`
	//Data interface{} `json:"data"`
}
type Net_interfaces struct {
	Context      string         `json:"context"`
	Network_card []network_card `json:"network_card"`
}
type Status_Detail struct {
	Bcode        string `json:"bcode"`
	Email        string `json:"email"`
	Node_version string `json:"node_version"`
	//K8sVersion string `json:"k8s_version"`
	Tun0 bool `json:"tun0"`
}

// Unmarshal used
type nodedb struct {
	Bcode      string `json:"bcode"`
	Email      string `json:"email"`
	Macaddress string `json:"macaddress"`
}

const install_ppp_script = `
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

var Don_ins_node bool
var Don_update bool

func main() {
	Init()
	go onboot()
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	e := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://console.bonuscloud.io",
		"http://bm.zzk2.icu", "http://127.0.0.1:8080", "http://localhost:8080"}
	//config.AllowAllOrigins = true
	e.Use(cors.New(config))
	e.GET("/discovery", tp_all)
	e.GET("/status", tp_all)
	e.POST("/bound", tp_all)
	e.POST("/disk", tp_all)
	disk := e.Group("/disk")
	disk.GET("/all", get_disk_all)
	e.GET("/version", tp_all)
	status := e.Group("/status")
	status.GET("/detail", get_status_detail)

	e.GET("/pppoe", get_ppp)
	e.POST("/pppoe", set_ppp)
	e.DELETE("/pppoe/:name", del_ppp)
	e.PATCH("/pppoe/:name",start_ppp)
	e.PATCH("/pppoe/:name/stop",stop_ppp)
	e.GET("/net", get_net)
	e.PATCH("/net", apply_net)
	e.PUT("/net", set_net)
	e.Run(":9018")

}

func Init() {
	flag.BoolVar(&Don_ins_node, "D", false, "Don install bxc-node")
	flag.BoolVar(&Don_update, "U", false, "Don check update")
	flag.Parse()
}

//// Transparent 透传至官方客户端
//func tp_discovery(c *gin.Context) {
//	Transparent("127.0.0.1:9017", c)
//}
//func tp_status(c *gin.Context) {
//	Transparent("127.0.0.1:9017", c)
//}
//func tp_bound(c *gin.Context) {
//	Transparent("127.0.0.1:9017", c)
//}
//func tp_disk(c *gin.Context) {
//	Transparent("127.0.0.1:9017", c)
//}
func tp_all(c *gin.Context) {
	Transparent("127.0.0.1:9017", c)
}

func Transparent(target string, c *gin.Context) {
	// target: target ip:port
	t := c.Request.URL
	t.Host = target
	reverseProxy := httputil.NewSingleHostReverseProxy(t)
	reverseProxy.Director = func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = target
		req.Host = target
	}
	reverseProxy.ServeHTTP(c.Writer, c.Request)
}

func get_status_detail(c *gin.Context) {
	//var err error
	var detail Status_Detail
	if PathExist("/opt/bcloud/node.db") {
		bt, err := ioutil.ReadFile("/opt/bcloud/node.db")
		if err != nil {
			log.Printf("get node.db fail: %s", err)
		} else {
			var node nodedb
			if err := json.Unmarshal(bt, &node); err != nil {
				log.Printf("Unmarshal node.db error: %s", err)
			} else {
				detail.Bcode = node.Bcode
				detail.Email = node.Email
			}
		}
	}
	var tmp_version map[string]string
	node_version_bt, err := GET("http://localhost:9017/version")
	if err == nil {
		err = json.Unmarshal(node_version_bt, &tmp_version)
		if err == nil {
			detail.Node_version = tmp_version["version"]
		} else {
			log.Printf("Unmarshal version fail:%s", err)
		}
	}
	if net_tun0, err := net.InterfaceByName("tun0"); err == nil {
		log.Println(net_tun0.Addrs())
		detail.Tun0 = true
	} else {
		detail.Tun0 = false
	}
	c.JSON(http.StatusOK, detail)
}

func get_ppp(c *gin.Context) {
	//name:=c.Params("name")
	acc := Read_dsl_file()
	c.JSON(http.StatusOK, acc)

}
func set_ppp(c *gin.Context) {
	var acc_conf Pppoe_account
	if err := c.ShouldBindJSON(&acc_conf); err != nil {
		log.Printf("bind to Pppoe_account error: %s", err)
	}
	if acc_conf.Name == "" {
		c.JSON(http.StatusNoContent, Message{http.StatusNoContent, "Not have name"})
	} else if acc_conf.Username == "" || acc_conf.Password == "" {
		c.JSON(http.StatusNoContent, Message{http.StatusNoContent, "Not have name"})
	}
	err := setppp(acc_conf)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			Message{http.StatusInternalServerError, "Set ppp account error:" +
				fmt.Sprintf("%s", err)})
	} else if err = run_ppp(acc_conf); err != nil {
		c.JSON(http.StatusOK, Message{http.StatusInternalServerError,
			fmt.Sprintf("pppoe call fail: %s", err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "Set " + acc_conf.Name + " OK"})
	}
}
func del_ppp(c *gin.Context) {
	name:=c.Param("name")
	if name=="" {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "resolve json failed"})
	}

	if ! PathExist("/etc/ppp/peers/" + name) {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("file %s not found", name)})
	}
	err := os.Remove("/etc/ppp/peers/" + name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("file %s remove failed", name)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, fmt.Sprintf("remove %s OK", name)})
	}
}
func start_ppp(c *gin.Context)  {
	filename:=c.Param("name")
	if filename=="" {
		c.JSON(http.StatusNotFound,Message{http.StatusNotFound,"not get a name"})
		return
	}
	if ! PathExist("/etc/ppp/peers/" + filename) {
		c.JSON(http.StatusNotFound,Message{http.StatusNotFound,"file name not found"})
	}
	kill_ppp(filename)
	if err:=run_ppp(Pppoe_account{filename,"","",ppp_conf{},false});err!=nil {
		c.JSON(http.StatusInternalServerError,Message{http.StatusInternalServerError,
		fmt.Sprintf("start pppoe file %s fail:%s",filename,err)})
	}
}
func stop_ppp(c *gin.Context)  {
	filename:=c.Param("name")
	if filename=="" {
		c.JSON(http.StatusNotFound,Message{http.StatusNotFound,"not get a name"})
		return
	}
	if ! PathExist("/etc/ppp/peers/" + filename) {
		c.JSON(http.StatusNotFound,Message{http.StatusNotFound,"file name not found"})
	}
	if err:=kill_ppp(filename);err!=nil {
		c.JSON(http.StatusInternalServerError,Message{http.StatusInternalServerError,
		fmt.Sprintf("stop %s fail: %s",filename,err)})
	}else {
		c.JSON(http.StatusOK,Message{http.StatusOK,"OK"})
	}
}
func get_net(c *gin.Context) {
	by, err := ioutil.ReadFile("/etc/network/interfaces")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("get file content fail: %s", err)})
		return
	}
	cards := get_network_card()
	c.JSON(http.StatusOK, Net_interfaces{string(by), cards})
}
func set_net(c *gin.Context) {
	var by Net_interfaces
	if err := c.ShouldBindJSON(&by); err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "resolve file failed"})
	}
	_, err := CopyFile("/etc/network/interfaces.bak", "/etc/network/interfaces")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("Copy file fail: %s", err)})
	}
	err = ioutil.WriteFile("/etc/network/interfaces", []byte(by.Context), 0644)
	if err != nil {

		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("Write file fail: %s", err)})
		_, err = CopyFile("/etc/network/interfaces", "/etc/network/interfaces.bak")
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
}
func apply_net(c *gin.Context) {
	cmd := exec.Command("systemctl", "restart", "networking")
	err := cmd.Start()
	if err != nil {
		log.Printf("run restart network fail: %s", err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Printf("run restart network fail: %s", err)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("run restart network fail: %s", err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
}
func get_disk_all(c *gin.Context) {
	block, err := hardware.Get_block()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("get block error: %s", err)})
	}
	c.JSON(http.StatusOK, block)
}

func setppp(p Pppoe_account) (error) {

	if err:=check_ppp();err!=nil{
		return err
	}
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
	if err := set_secrets(p, "/etc/ppp/chap-secrets"); err != nil {
		return err
	}
	return set_secrets(p, "/etc/ppp/pap-secrets")
}


func run_ppp(p Pppoe_account) (error) {

	cmd := exec.Command("pppd", "call", p.Name)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}
func kill_ppp(name string) error {
	return Run_command(fmt.Sprintf("kill -TERM `cat /var/run/ppp-%s.pid|head -n 1`",name))
}
func Get_ppp_status(p Pppoe_account) (Pppoe_account) {
	pid_file:=fmt.Sprintf("/var/run/ppp-%s.pid",p.Name)
	if ! PathExist(pid_file) {
		log.Printf("not found pid file: %s",pid_file)
		p.Status=false
		return p
	}
	content,err:=ioutil.ReadFile(pid_file)
	if err!=nil {
		log.Printf("read pid file %s fail: %s",pid_file,err)
		return p
	}
	spl:=strings.Split(string(content),"\n")
	log.Println(spl)
	pid:=spl[0]
	p.Status=(pid!="")
	return p
}
func Read_dsl_file() []Pppoe_account {
	configs := []Pppoe_account{}
	files := getFilelist("/etc/ppp/peers")
	if len(*files) == 0 {
		return []Pppoe_account{}
	}
	for _, c := range *files {
		tmp, err := resolve_dsl_file(c)
		if err != nil {
			log.Printf("resolve file %s fail", err)
			continue
		}

		configs = append(configs, *tmp)
	}
	configs = read_chap_secrets(configs)
	for i,p:=range configs {
		configs[i]=Get_ppp_status(p)
	}
	//log.Println(configs)
	return configs
}
func resolve_dsl_file(f_path string) (*Pppoe_account, error) {
	var p Pppoe_account
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
func getFilelist(path string) *[]string {
	files := &[]string{}
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if (f == nil) {
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

func read_chap_secrets(acc []Pppoe_account) []Pppoe_account {
	fc, err := os.Open("/etc/ppp/chap-secrets")
	if err != nil {
		log.Printf("read chap-secrets fail: %s", err)
		return nil
	}
	defer fc.Close()
	//acc:=[]Pppoe_account{}
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
		//tmp_p := Pppoe_account{}
		//tmp_p.Username = tmp_s_s[0]
		//tmp_p.Password = tmp_s_s[2]
		//acc = append(acc, tmp_p)
	}
	return acc
}
func set_secrets(p Pppoe_account, filename string) (error) {
	//filename:="/etc/ppp/chap-secrets"
	if ! PathExist(filename) {
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
			if strings.Contains(line, p.Username, ) {
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
	log.Print(string(fstr))
	_, err = fc.Write(fstr)
	return err
}

func get_network_card() []network_card {
	netIs, err := net.Interfaces()
	if err != nil {
		log.Printf("fail to get net interfaces: %v", err)
		return nil
	}

	net_cards := []network_card{}
	for _, netI := range netIs {
		tmp := network_card{}
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
func getip() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	defer conn.Close()
	log.Println(strings.Split(conn.LocalAddr().String(), ":")[0])
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}

func check_ppp() (error) {
	_, err := exec.LookPath("pppd")
	if err != nil {
		return Intsall_ppp()
	}
	if ! PathExist("/dev/ppp") {
		return Intsall_ppp()
	}
	return nil
}
func Intsall_ppp() error {
	if err := Run_command(install_ppp_script); err != nil {
		log.Printf("Install pppoe software failed")
		return err
	}
	return nil
}

func GET(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("get %s fail:%s", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read body fail:%s", err)
		return nil, err
	}
	return body, nil

}
