package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	Other     []string `json:"other"`
	Mtu       int      `json:"mtu"`
}
type Pppoe_account struct {
	Name     string   `json:"name"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Conf     ppp_conf `json:"conf"`
}
type network_card struct {
	Name    string   `json:"name"`
	Macaddr string   `json:"macaddr"`
	Ip      []string `json:"ip"`
}
type Message struct {
	Code    int    `json:"code"`
	Details string `json:"details"`
}
type Net_interfaces struct {
	Context      string         `json:"context"`
	Network_card []network_card `json:"network_card"`
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



func main() {
	go onboot()
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	e := gin.Default()
	config := cors.DefaultConfig()
	//config.AllowOrigins = []string{"https://console.bonuscloud.io", "https://ssl.ws.lan", "http://ssl.ws.lan","http://127.0.0.1"}
	config.AllowAllOrigins = true
	e.Use(cors.New(config))
	e.GET("/discovery", tp_all)
	e.GET("/status", tp_all)
	e.POST("/bound", tp_all)
	e.POST("/disk", tp_all)
	e.GET("/version", tp_all)

	e.GET("/pppoe", get_ppp)
	e.POST("/pppoe", set_ppp)
	e.DELETE("/pppoe", del_ppp)
	e.GET("/net", get_net)
	e.PATCH("/net", apply_net)
	e.PUT("/net", set_net)
	e.GET("/test", func(c *gin.Context) {
		c.JSON(200,Message{200,"this test api3"})
	})
	//s:=&http.Server{
	//	Addr: ":9018",
	//	Handler:e,
	//	//ReadTimeout:    10 * time.Second,
	//	//WriteTimeout:   10 * time.Second,
	//	//MaxHeaderBytes: 1 << 20,
	//}
	e.Run(":9018")
	//gracehttp.AddServer(s,false,"","")
	//gracehttp.Run()

}
// Transparent 透传至官方客户端
func tp_discovery(c *gin.Context) {
	Transparent("127.0.0.1:9017", c)
}
func tp_status(c *gin.Context) {
	Transparent("127.0.0.1:9017", c)
}
func tp_bound(c *gin.Context) {
	Transparent("127.0.0.1:9017", c)
}
func tp_disk(c *gin.Context) {
	Transparent("127.0.0.1:9017", c)
}
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



func get_ppp(c *gin.Context) {
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
	var p Pppoe_account
	if err := c.ShouldBindJSON(&p); err != nil {
		log.Printf("bind to Pppoe_account error: %s", err)
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "resolve json failed"})
	}
	if p.Name == "" {
		c.JSON(http.StatusNoContent, Message{http.StatusNoContent, "Not have name"})
	}
	if ! PathExist("/etc/ppp/peers/" + p.Name) {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("file %s not found", p.Name)})
	}
	err := os.Remove("/etc/ppp/peers/" + p.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("file %s remove failed", p.Name)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, fmt.Sprintf("remove %s OK", p.Name)})
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

func setppp(p Pppoe_account) (error) {
	fs, err := os.OpenFile("/etc/ppp/peers/"+p.Name, os.O_WRONLY|os.O_CREATE, 0660)
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
	_, err = fs.WriteString(conf_str)
	if err != nil {
		return err
	}
	err = set_chap_secrets(p)
	return err
}
func run_ppp(p Pppoe_account) (error) {
	cmd := exec.Command("pppd", "call", p.Name)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
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
	log.Println(configs)
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
func set_chap_secrets(p Pppoe_account) (error) {
	fstr, err := ioutil.ReadFile("/etc/ppp/chap-secrets")
	if err != nil {
		log.Println("set passwd fail", err)
		return err
	}
	fc, err := os.OpenFile("/etc/ppp/chap-secrets", os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0600)
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

