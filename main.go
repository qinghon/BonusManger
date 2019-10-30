package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/qinghon/hardware"
	"github.com/qinghon/network"
	"github.com/qinghon/system/bonus"
	"github.com/qinghon/system/tools"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"time"
)

const Version = "v0.3.11"

type Message struct {
	Code    int    `json:"code"`
	Details string `json:"details"`
	//Data interface{} `json:"data"`
}
type StatusDetail struct {
	Bcode       string `json:"bcode"`
	Email       string `json:"email"`
	NodeVersion string `json:"node_version"`
	//K8sVersion string `json:"k8s_version"`
	Tun0 bool `json:"tun0"`
}

// Unmarshal used
type nodedb struct {
	Bcode      string `json:"bcode" binding:"required"`
	Email      string `json:"email"`
	Macaddress string `json:"macaddress"`
}

var DonInsNode bool
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
	e.GET("/discovery", tpAll)
	e.GET("/status", tpAll)
	e.GET("/version", tpAll)
	status := e.Group("/status")
	{
		status.GET("/detail", getStatusDetail)
	}
	e.POST("/bound", tpAll)
	e.POST("/disk", tpAll)
	disk := e.Group("/disk")
	{
		disk.GET("/all", getDiskAll)
		disk.POST("/umount/:part", umountPart)
		disk.POST("/format", formatPart)
		native := disk.Group("/native")
		{
			native.GET("/:device", middleDiskNative, getDiskNative)
			native.POST("/:device", middleDiskNative, setDiskNative)
			native.DELETE("/:device/:part", middleDiskNative, delDiskNativePart)
		}
		lvm := disk.Group("/lvm")
		{
			lvm.GET("/lv", getLv)
			lvm.POST("/lv")
			lvm.DELETE("/lv", delLv)

			lvm.GET("/vg", getVg)
			lvm.POST("/vg", createVg)
			lvm.DELETE("/vg/:name", delVg)

			lvm.GET("/pv", getPv)
			lvm.POST("/pv", createPv)
			lvm.DELETE("/pv/:name")
		}
	}

	e.GET("/pppoe", getPpp)
	e.POST("/pppoe", setPpp)
	e.POST("/pppoe/install", installPpp)
	e.DELETE("/pppoe/:name", delPpp)
	e.PATCH("/pppoe/:name", startPpp)
	e.PATCH("/pppoe/:name/stop", stopPpp)
	e.GET("/net", getNet)
	e.PATCH("/net", applyNet)
	e.PUT("/net", setNet)
	e.POST("/bonus/repair", repair)
	e.POST("/update", update)
	//e.GET("/system/log",getLog)
	tool := e.Group("/tools")
	{
		tool.GET("/reboot", reboot)
		tool.GET("/shutdown", shutdown)
		tool.POST("/ssh", openssh)
	}
	e.GET("/v", getVersion)
	e.Run(":9018")

}

func Init() {
	flag.BoolVar(&DonInsNode, "D", false, "Don install bxc-node")
	flag.BoolVar(&Don_update, "U", false, "Don check update")
	var v = flag.Bool("V", false, "show version")
	flag.Parse()
	if *v {
		showVersion()
	}
}

/*// transparent 透传至官方客户端
func tp_discovery(c *gin.Context) {
	transparent("127.0.0.1:9017", c)
}
func tp_status(c *gin.Context) {
	transparent("127.0.0.1:9017", c)
}
func tp_bound(c *gin.Context) {
	transparent("127.0.0.1:9017", c)
}
func tp_disk(c *gin.Context) {
	transparent("127.0.0.1:9017", c)
}*/
/* transparent 透传至官方客户端 */
func tpAll(c *gin.Context) {
	transparent("127.0.0.1:9017", c)
}

func transparent(target string, c *gin.Context) {
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

func getStatusDetail(c *gin.Context) {
	//var err error
	var detail StatusDetail
	if network.PathExist(bonus.NODEDB) {
		bt, err := ioutil.ReadFile(bonus.NODEDB)
		if err != nil {
			log.Printf("get node.db fail: %s", err)
		} else {
			var node nodedb
			if err := json.Unmarshal(bt, &node); err != nil {
				log.Printf("Unmarshalb node.db error: %s", err)
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
			detail.NodeVersion = tmp_version["version"]
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

func getPpp(c *gin.Context) {
	//name:=c.Params("name")
	acc := network.ReadDslFile()
	c.JSON(http.StatusOK, acc)

}
func setPpp(c *gin.Context) {
	var acc_conf network.PppoeAccount
	if err := c.ShouldBindJSON(&acc_conf); err != nil {
		log.Printf("bind to PppoeAccount error: %s", err)
	}
	if acc_conf.Name == "" {
		c.JSON(http.StatusNoContent, Message{http.StatusNoContent, "Not have name"})
	} else if acc_conf.Username == "" || acc_conf.Password == "" {
		c.JSON(http.StatusNoContent, Message{http.StatusNoContent, "Not have name"})
	}
	err := network.Setppp(acc_conf)
	if err != nil {
		c.JSON(http.StatusNotImplemented,
			Message{http.StatusNotImplemented, "Set ppp account error:" +
				fmt.Sprintf("%s", err)})
	} else if by, err := network.RunPpp(acc_conf); err != nil {
		c.JSON(http.StatusOK, Message{http.StatusInternalServerError,
			fmt.Sprintf("pppoe call fail: %s\n %s", string(by), err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "Set " + acc_conf.Name + " OK "})
	}
}
func delPpp(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "resolve json failed"})
	}
	if !network.PathExist("/etc/ppp/peers/" + name) {
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
func installPpp(c *gin.Context) {
	if by, err := network.CheckPpp(); err != nil {
		c.JSON(http.StatusOK, Message{http.StatusInternalServerError,
			fmt.Sprintf("pppoe install fail: \n %s\n %s", string(by), err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
}
func startPpp(c *gin.Context) {
	filename := c.Param("name")
	if filename == "" {
		c.JSON(http.StatusNotFound, Message{http.StatusNotFound, "not get a name"})
		return
	}
	if !network.PathExist("/etc/ppp/peers/" + filename) {
		c.JSON(http.StatusNotFound, Message{http.StatusNotFound, "file name not found"})
	}
	network.KillPpp(filename)
	if by, err := network.RunPpp(network.PppoeAccount{filename, "", "", network.PppConf{}, network.PppStatus{}}); err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("start pppoe file %s fail:%s\n%s", filename, string(by), err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, fmt.Sprintf("%s", string(by))})
	}
}
func stopPpp(c *gin.Context) {
	filename := c.Param("name")
	if filename == "" {
		c.JSON(http.StatusNotFound, Message{http.StatusNotFound, "not get a name"})
		return
	}
	if !network.PathExist("/etc/ppp/peers/" + filename) {
		c.JSON(http.StatusNotFound, Message{http.StatusNotFound, "file name not found"})
	}
	if err := network.KillPpp(filename); err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("stop %s fail: %s", filename, err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
}
func getNet(c *gin.Context) {
	by, err := ioutil.ReadFile("/etc/network/interfaces")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("get file content fail: %s", err)})
		return
	}
	cards := network.GetNetworkCard()
	c.JSON(http.StatusOK, network.NetInterfaces{string(by), cards})
}
func setNet(c *gin.Context) {
	var by network.NetInterfaces
	if err := c.ShouldBindJSON(&by); err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "resolve file failed"})
	}
	_, err := network.CopyFile("/etc/network/interfaces.bak", "/etc/network/interfaces")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("Copy file fail: %s", err)})
	}
	err = ioutil.WriteFile("/etc/network/interfaces", []byte(by.Context), 0644)
	if err != nil {

		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("Write file fail: %s", err)})
		_, err = network.CopyFile("/etc/network/interfaces", "/etc/network/interfaces.bak")
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
}
func applyNet(c *gin.Context) {
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
func getDiskAll(c *gin.Context) {
	block, err := hardware.Get_block()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Message{http.StatusServiceUnavailable,
			fmt.Sprintf("get block error: %s", err)})
	}
	c.JSON(http.StatusOK, block)
}
func update(c *gin.Context) {
	file, err := c.FormFile("bonusmanger")
	if err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest,
			fmt.Sprintf("not found upload file:%s", err)})
	}
	log.Printf("load upload update exec")
	fp, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("open file failed: %s", err)})
	}
	content, err := ioutil.ReadAll(fp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("read file failed: %s", err)})
	}
	log.Printf("file size : %d", len(content))
	f, err := os.OpenFile("/tmp/bonusmanger", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	defer f.Close()
	_, err = f.Write(content)
	if err != nil {
		log.Printf("write file failed:  %s", err)
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("write file failed:  %s", err)})
		return
	}
	err = CopyfileForce("/opt/BonusManger/bin/bonusmanger", "/tmp/bonusmanger")
	if err != nil {
		log.Printf("copy file failed:  %s", err)
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("copy file failed:  %s", err)})
	}
	c.JSON(http.StatusOK, Message{http.StatusOK, "OK ,reboot now"})
	go func() {
		time.Sleep(time.Second * 2)
		os.Exit(1)
	}()
}
func getVersion(c *gin.Context) {
	md5sumLocal := Getfilemd5(os.Args[0])
	c.JSON(http.StatusOK, gin.H{"version": Version, "md5sum": md5sumLocal})
}

func repair(c *gin.Context) {
	if _, err := bonus.ReadNodedb(); err == nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError, "证书及描述文件已存在"})
		return
	}
	var db nodedb
	if err := c.ShouldBindJSON(&db); err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "resolve file false"})
		return
	}
	if db.Email == "" {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "email or bcode is null"})
		return
	}
	bcode, err := bonus.ReadBcode()
	if err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest,
			fmt.Sprintf("read bcode error:%s", err)})
		return
	}
	db.Bcode = bcode
	js, err := json.Marshal(map[string]string{"bcode": db.Bcode, "email": db.Email})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError, "json encode fail"})
		return
	}
	if err := ioutil.WriteFile(bonus.NODEDB, js, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError, "write fail"})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
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

func formatPart(c *gin.Context) {
	var form struct {
		Dev  string `json:"dev" binding:"required"`
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest,
			fmt.Sprintf("Wrong parameter: %s ", err)})
	}
	var p hardware.Partition
	p.Name = form.Dev
	by, err := p.Format(form.Dev)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("Format part % fail:%s;%s", form.Dev, err, string(by))})
	} else {
		c.JSON(http.StatusOK, "OK")
	}
}
func getLv(c *gin.Context) {
	lv, err := hardware.GetLv()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("get lv error: %s", err)})
		return
	}
	c.JSON(http.StatusOK, lv.Report[0].Lv)
}
func delLv(c *gin.Context) {
	vg := c.Query("vg")
	lv := c.Query("lv")
	if vg == "" || lv == "" {
		c.JSON(http.StatusNotFound, Message{http.StatusNotFound, "not get vg name or lv name"})
		return
	}
	var lvs hardware.Lv
	lvs.LvName = lv
	lvs.VgName = vg
	lvinfo, err := hardware.RemoveLv([]hardware.Lv{lvs})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("delete lv fail:%s", err)})
	} else {
		c.JSON(http.StatusOK, lvinfo)
	}
}

func getPv(c *gin.Context) {
	pv, err := hardware.GetPv()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("get pv error: %s", err)})
		return
	}
	c.JSON(http.StatusOK, pv.Report[0].Pv)
}
func createPv(c *gin.Context) {
	var formPv struct {
		PvName string `json:"pv_name"` //device name
		VgName string `json:"vg_name"`
	}
	if err := c.ShouldBindJSON(&formPv); err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest,
			fmt.Sprintf("Wrong parameter: %s ", err)})
	}
	_, err := hardware.CreatePV(formPv.PvName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("Create vg fail:%s", err)})
	}
	if formPv.VgName == "" {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
		return
	}
	vginfo, err := hardware.ExtendVg(formPv.PvName, formPv.VgName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("extend vg fail:%s ,but format pv success", err)})
	} else {
		c.JSON(http.StatusOK, vginfo)
	}
}

/*func reducePv(c *gin.Context) {

}*/
func getVg(c *gin.Context) {
	vg, err := hardware.GetVg()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("get vg error: %s", err)})
		return
	}
	c.JSON(http.StatusOK, vg.Report[0].Vg)
}
func createVg(c *gin.Context) {
	var vg struct {
		VgName string   `json:"vg_name" binding:"required"`
		PvName []string `json:"pv_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&vg); err != nil || vg.PvName == nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest,
			fmt.Sprintf("Wrong parameter: %s ", err)})
	}
	vginfo, err := hardware.CreateVg(vg.VgName, vg.PvName...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("Create vg fail:%s", err)})
	} else {
		c.JSON(http.StatusOK, vginfo)
	}
}
func delVg(c *gin.Context) {
	vg := c.Param("name")
	if vg == "" {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "not get empty device"})
		return
	}
	var Vg hardware.Vg
	Vg.VgName = vg
	vginfo, err := hardware.RemoveVg(Vg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("delete vg fail:%s", err)})
	} else {
		c.JSON(http.StatusOK, vginfo)
	}
}

func middleDiskNative(c *gin.Context) {
	d := c.Param("device")
	if d == "" {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "not get empty device"})
		return
	}
	c.Next()
}
func getDiskNative(c *gin.Context) {
	d := c.Param("device")
	disk, err := hardware.DiskInfo(d)
	log.Println(disk.Table)
	if err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest,
			fmt.Sprintf("get disk info fail:%s", err)})
	} else {
		c.JSON(http.StatusOK, disk)
	}
}
func setDiskNative(c *gin.Context) {
	d := c.Param("device")
	disk, err := hardware.DiskInfo(d)
	var p hardware.Partition
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "params not true"})
		return
	}
	by, err := disk.CreatePart(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("create partition fail: %s", by)})
	}
	disk, err = hardware.DiskInfo(d)
	c.JSON(http.StatusOK, disk)
}
func delDiskNativePart(c *gin.Context) {
	var disk hardware.Dev
	d := c.Param("device")
	disk.Name = d
	p := c.Param("part") // It's number
	if p == "" {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "not get partition"})
		return
	}
	by, err := disk.DeletePart(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("delete partition fail: %s", by)})
	}
	disk, err = hardware.DiskInfo(d)
	c.JSON(http.StatusOK, disk)
}
func umountPart(c *gin.Context) {
	p := c.Param("part") //It's part name like: sda1
	if p == "" {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "not get partition"})
	}
	if err := hardware.UmountDev(p); err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("umount part fail:%s", err)})
	}
}

func shutdown(c *gin.Context) {
	if err := tools.Shutdown(); err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("shutdown fail:%s", err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
}
func reboot(c *gin.Context) {
	//go func() {
	//	time.Sleep(time.Second*5)
	//	syscall.Reboot(0)
	//}()
	if err := tools.Reboot(); err != nil {
		c.JSON(http.StatusInternalServerError, Message{http.StatusInternalServerError,
			fmt.Sprintf("reboot fail:%s", err)})
	} else {
		c.JSON(http.StatusOK, Message{http.StatusOK, "OK"})
	}
}
func openssh(c *gin.Context) {
	var k tools.Key
	if err := c.ShouldBindJSON(&k); err != nil {
		c.JSON(http.StatusBadRequest, Message{http.StatusBadRequest, "params not true"})
		return
	}
	clip := c.ClientIP()
	ipt := net.ParseIP(clip)
	var isPrivate bool
	netcard := network.GetNetworkCard()
	if len(netcard) == 0 {
		isPrivate = false
	}
	for _, c := range netcard {
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
	if !isPrivate {
		c.JSON(http.StatusForbidden, Message{http.StatusForbidden,
			"you need in private network set this"})
	}
	if len(k.PublicKey) == 0 {
		k.GenKey(2048)
	}
	if err := k.Trust(); err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("Trust ssh fail: %s", err))
	} else {
		c.JSON(http.StatusOK, k)
	}
	k.PrivateKey = ""
}

func showVersion() {
	fmt.Print(Version)
	os.Exit(0)
}
