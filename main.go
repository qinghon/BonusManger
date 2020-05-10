package main

import (
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"strings"
)

const Version = "v0.4.4"

type Message struct {
	Code    int    `json:"code"`
	Details string `json:"details"`
	//Data interface{} `json:"data"`
}

// Unmarshal used
type nodedb struct {
	Bcode      string `json:"bcode" binding:"required"`
	Email      string `json:"email"`
	Macaddress string `json:"macaddress"`
}

var DonInsNode bool
var DonUpdate bool
var logLevel int
var debug bool
var NoSetOptions bool

var config Config

func main() {
	Init()
	go onboot()

	e := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
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
	/*
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
			lvm.POST("/lv", createLv)
			lvm.DELETE("/lv", delLv)

			lvm.GET("/vg", getVg)
			lvm.POST("/vg", createVg)
			lvm.DELETE("/vg/:name", delVg)

			lvm.GET("/pv", getPv)
			lvm.POST("/pv", createPv)
			lvm.DELETE("/pv", delPv)
		}
	}
	*/

	e.GET("/pppoe", getPpp)
	e.POST("/pppoe", setPpp)
	e.POST("/pppoe/install", installPpp)
	e.DELETE("/pppoe/:name", delPpp)
	e.PATCH("/pppoe/:name", startPpp)
	e.PATCH("/pppoe/:name/stop", stopPpp)
	e.GET("/pppoe/:name/log", getPppLog)
	e.GET("/net", getNet)
	e.PATCH("/net", applyNet)
	e.PUT("/net", hardSetNet)
	//netApi := e.Group("/net", )
	//{
	//	netApi.GET("/eth")
	//	netApi.POST("/eth")
	//	netApi.GET("/ppp")
	//	netApi.POST("/ppp")
	//}
	b := e.Group("/bonus")
	{
		b.POST("/repair", repair)
		b.GET("/status", bonusGetStatus)
	}
	e.POST("/update", update)
	//e.GET("/system/log",getLog)
	tool := e.Group("/tools")
	{
		tool.GET("/reboot", reboot)
		tool.GET("/shutdown", checkPrivateIp, shutdown)
		tool.POST("/ssh", checkPrivateIp, openssh)
		tool.GET("/ws", checkPrivateIp, WsSsh)
		tool.GET("/remarks", getRemarks)
		tool.POST("/remarks", setRemarks)
	}
	e.GET("/v", getVersion)
	e.Run(":9018")
}

func Init() {
	flag.BoolVar(&DonInsNode, "D", false, "Don install bxc-node. ")
	flag.BoolVar(&DonUpdate, "U", false, "Don check update. ")
	flag.IntVar(&logLevel, "level", 4, "Show log level. ")
	flag.BoolVar(&debug, "debug", false, "Debug mode. ")
	var v = flag.Bool("no-set-options", false, "no set ppp options file: /etc/ppp/options .")
	flag.Parse()
	if *v {
		showVersion()
	}

	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			s := strings.Split(f.Function, ".")
			funcName := s[len(s)-1]
			return funcName, fmt.Sprintf(" %s:%d", path.Base(f.File), f.Line)
		},
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.Level(logLevel))
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		return
	}
	err := config.get()
	if err != nil {
		logrus.Error(err)
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
func showVersion() {
	fmt.Print(Version)
	os.Exit(0)
}
