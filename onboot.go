package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/qinghon/network"
	"github.com/qinghon/system/bonus"
	"github.com/qinghon/system/tools"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const VersionURLS = "https://github.com/qinghon/BonusManger/releases/download/%s/md5sum"

const GetClient = "https://github.com/qinghon/BonusManger/releases/download/%s/bonus_manger_%s"
const BxcNodeURL = "https://github.com/BonusCloud/BonusCloud-Node/raw/master/img-modules/bxc-node_%s"
const BxcNodeService = `
[Unit]
Description=bxc node app
After=network.target

[Service]
ExecStart=/opt/bcloud/nodeapi/node --alsologtostderr
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`

const githubLatest = "https://api.github.com/repos/qinghon/BonusManger/releases/latest"

var ARCH string
var lastReleaseData releaseLatest

/*  Auto Generated*/
type releaseLatest struct {
	URL             string `json:"url"`
	AssetsURL       string `json:"assets_url"`
	UploadURL       string `json:"upload_url"`
	HTMLURL         string `json:"html_url"`
	ID              int    `json:"id"`
	NodeID          string `json:"node_id"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Name            string `json:"name"`
	Draft           bool   `json:"draft"`
	Author          struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		URL      string      `json:"url"`
		ID       int         `json:"id"`
		NodeID   string      `json:"node_id"`
		Name     string      `json:"name"`
		Label    interface{} `json:"label"`
		Uploader struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"uploader"`
		ContentType        string    `json:"content_type"`
		State              string    `json:"state"`
		Size               int       `json:"size"`
		DownloadCount      int       `json:"download_count"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		BrowserDownloadURL string    `json:"browser_download_url"`
	} `json:"assets"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
	Body       string `json:"body"`
}

func onboot() {
	go setArch()
	//go network.PatchPpp()
	go StartPPP()
	if !DonInsNode {
		go checkNode()
	} else {
		log.Println("not install as command flag")
	}
	if DonUpdate {
		log.Println("Ok..,it looks like you don't want to upgrade,I got it")
		return
	}
	timetkm := time.NewTicker(time.Minute * 20)
	//timetkp:=time.NewTicker(time.Hour)
	for {
		select {
		case <-timetkm.C:
			go checkAndUpdate()
		//case <-timetkp.C:
		//	network.CLOSE_TK<-false
		//	go network.CheckLinkAll()
		}
	}
}
func checkVersion() (string, bool) {
	var err error
	lastReleaseData, err = getLatestInfo()
	if err != nil {
		log.Printf("get tag info fail:%s", err)
		return "", false
	}
	if lastReleaseData.TagName == "" {
		log.Println("not found new tag")
		return "", false
	}
	//log.Println(fmt.Sprintf(VersionURLS, lastReleaseData.TagName))
	resp, err := http.Get(fmt.Sprintf(VersionURLS, lastReleaseData.TagName))
	if err != nil {
		log.Printf("get version failed: %s", err)
		return "", false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		println("read body failed")
	}
	spl := strings.Split(string(body), "\n")
	md5sumLocal := Getfilemd5(os.Args[0])
	for _, l := range spl {
		if strings.TrimSpace(l) == "" {
			continue
		}
		splLS := strings.Split(l, " ")
		for i, splLC := range splLS {
			//log.Println(splLC)
			if splLC == "" {
				splLS = append(splLS[:i], splLS[i+1:]...)
			}
		}
		md5sum := splLS[0]
		filename := splLS[1]
		//log.Println(ARCH, filename, strings.Contains(ARCH, filename), splLS, len(splLS))
		if !strings.Contains(filename, ARCH) {
			continue
		}
		//log.Println(md5sum, md5sumLocal)
		if md5sum != md5sumLocal {

			return md5sum, true
		} else {
			return "", false
		}
		//log.Println(md5sum,filename)
	}
	return "", false
}

func checkAndUpdate() {
	md5sum, needUpdate := checkVersion()
	//log.Println(md5sum)
	if needUpdate {
		log.Printf("we need update to %s,new client md5sum: %s", lastReleaseData.TagName, md5sum)
		downNewClient(md5sum)
	} else {
		log.Println("don't need update")
	}
}

func downNewClient(md5sum string) {
	downPath := "/tmp/bouns_manger"
	err := DownloadFileWget(fmt.Sprintf(GetClient, lastReleaseData.TagName, ARCH), downPath)
	if err != nil {
		log.Printf("Download new client fail: %s", err)

		err = DownloadFile(fmt.Sprintf(GetClient, lastReleaseData.TagName, ARCH), downPath)
		if err != nil {
			log.Printf("Download new client fail: %s", err)
		}
	}
	if md5sum == "" {
		err = CopyForce(os.Args[0], downPath)
		if err != nil {
			log.Printf("Copy to %s failed:%s", os.Args[0], err)
		}
	} else {
		downEdMd5 := Getfilemd5(downPath)
		if downEdMd5 != md5sum {
			log.Printf("down load file md5sum:%s not equal get file md5sum:%s", downEdMd5, md5sum)
			return
		} else {
			err = CopyForce(os.Args[0], downPath)
		}
	}
	if err := os.Chmod(os.Args[0], 0755); err != nil {
		log.Printf("chmod failed %s", downPath)
		return
	}
	log.Println("restarting server...,if this not restart, you should be run :\nsystemctl start bonus_manger")
	os.Exit(1)
}

func checkNode() error {
	_, err := http.Get("http://127.0.0.1:9017/discovery")
	if err == nil {
		log.Println("bxc-node may be is running")
		return err
	}
	err = os.MkdirAll("/opt/bcloud/nodeapi/", 0755)
	if err != nil {
		log.Printf("mkdir  /opt/bcloud/nodeapi/ fail: %s", err)
		return err
	}
	DownloadFile(fmt.Sprintf(BxcNodeURL, ARCH), "/opt/bcloud/nodeapi/node")
	err = os.Chmod("/opt/bcloud/nodeapi/node", 0755)
	if err != nil {
		return err
	}
	if !tools.PathExist(bonus.NODEDB) {
		_, err = os.Create(bonus.NODEDB)
		if err != nil {
			return err
		}
	}
	ioutil.WriteFile("/lib/systemd/system/bxc-node.service", []byte(BxcNodeService), 0644)
	if err != nil {
		return err
	}
	cmd := exec.Command("sh", "-c", "systemctl enable bxc-node&&systemctl start bxc-node")
	return cmd.Wait()

}

func Getfilemd5(_path string) string {
	f, err := os.Open(_path)
	if err != nil {
		log.Printf("Open file err: %s", err)
		return ""
	}
	defer f.Close()
	md5h := md5.New()
	io.Copy(md5h, f)
	hashstr := md5h.Sum([]byte(""))
	return fmt.Sprintf("%x", hashstr)
}

func DownloadFile(_URL, _path string) error {
	resp, err := http.Get(_URL)
	if err != nil {
		log.Printf("get file error:%s ,url: %s", err, _URL)
		return err
	}
	defer resp.Body.Close()
	outfile, err := os.Create(_path)
	if err != nil {
		log.Printf("Create file  failed: error: %s", err)
		return err
	}
	defer outfile.Close()
	_, err = io.Copy(outfile, resp.Body)
	if err != nil {
		log.Printf("write file fail: %s", err)
		return err
	}
	return nil
}
func DownloadFileWget(_URL, _path string) error {
	_, err := exec.LookPath("wget")
	if err != nil {
		return err
	}
	_ = os.Remove(_path)
	cmd := exec.Command("wget", "-c", "-O", _path, _URL)
	_, err = tools.CmdStdout(cmd)
	return err
}
func CopyForce(dstName, srcName string) error {
	if tools.PathExist(dstName) {
		if err := syscall.Unlink(dstName); err != nil {
			return err
		}
	}
	srcFile, _ := os.Open(srcName)
	dstFile, err := os.OpenFile(dstName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = io.Copy(srcFile, dstFile)
	if err != nil {
		log.Println(err)
	}
	defer srcFile.Close()
	defer dstFile.Close()
	return err
}
func setArch() {
	switch runtime.GOARCH {
	case "amd64":
		ARCH = "x86_64"
	case "arm64":
		ARCH = "aarch64"
	case "arm":
		ARCH = "armv7l"
	}
	log.Printf("check device arch is: %s", ARCH)
}

func getLatestInfo() (releaseLatest, error) {
	resp, err := http.Get(githubLatest)
	if err != nil {
		log.Printf("get latest tag fail: %s", err)
		return releaseLatest{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	var release releaseLatest
	err = json.Unmarshal(body, &release)
	if err != nil {
		log.Printf("Unmarshal fail: %s", err)
		return releaseLatest{}, err
	}
	lastReleaseData = release
	return release, nil

}
func StartPPP() {
	network.PPP_POOL=make(map[string]*network.StatusStack)
	network.CMD_CHAN=make(chan *exec.Cmd,5)
	pas:=network.ReadDslFile()
	go func() {
		var cmd *exec.Cmd
		for {
			select {
			case cmd = <-network.CMD_CHAN:
				log.Debug("get chan cmd")
				go runPPP(cmd)
				log.Debug("cmd started!")
			}
		}
		log.Debug("exited select")
	}()
	for _,pa:=range pas {

		err:=pa.Connect()
		if err != nil {
			log.Println(err)
		}

	}
	//time.Sleep(time.Second*5)
	//go network.CheckLinkAll()
	//pppd "nodetach" "noipdefault" "defaultroute" "hide-password" "noauth" "persist" "plugin" "rp-pppoe.so" "maxfail" "0" user test1 password 123456 "lcp-echo-failure" "4" "lcp-echo-interval" "30" "linkname" "test1" "eth0"  logfile /var/run/pppd.log
	var wait chan int
	<-wait
	log.Debug("Useful debugging information.")
	log.Debug("all pppd exited!")

}
func runPPP(cmd *exec.Cmd) ([]byte,error) {
	return tools.CmdStdout(cmd)
}