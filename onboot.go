package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"system/tools"
	"time"
)

const VersionURLS = "https://github.com/qinghon/BonusManger/releases/download/%s/md5sum"

const GetClient = "https://github.com/qinghon/BonusManger/releases/download/%s/bonus_manger_%s"
const Bxc_node_URL = "https://github.com/BonusCloud/BonusCloud-Node/raw/master/img-modules/bxc-node_%s"
const Bxc_node_service = `
[Unit]
Description=bxc node app
After=network.target

[Service]
ExecStart=/opt/bcloud/nodeapi/node --alsologtostderr ${DON_SET_DISK} ${INSERT_STR} 
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`

const Github_latest = "https://api.github.com/repos/qinghon/BonusManger/releases/latest"

var ARCH string
var last_release releaseLatest

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
	go Set_arch()
	if !DonInsNode {
		go check_node()
	} else {
		log.Println("not install as command flag")
	}
	if Don_update {
		log.Println("Ok..,it looks like you don't want to upgrade,I got it")
		return
	}
	timetkm := time.NewTicker(time.Minute * 10)
	for {
		select {
		case <-timetkm.C:
			go check_and_update()
		}
	}
}
func check_version() (string, bool) {
	var err error
	last_release, err = getLatestInfo()
	if err != nil {
		log.Printf("get tag info fail:%s", err)
		return "", false
	}
	if last_release.TagName == "" {
		log.Println("not found new tag")
		return "", false
	}
	//log.Println(fmt.Sprintf(VersionURLS, last_release.TagName))
	resp, err := http.Get(fmt.Sprintf(VersionURLS, last_release.TagName))
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
	md5sum_local := Getfilemd5(os.Args[0])
	for _, l := range spl {
		if strings.TrimSpace(l) == "" {
			continue
		}
		spl_l := strings.Split(l, " ")
		for i, spl_l_c := range spl_l {
			//log.Println(spl_l_c)
			if spl_l_c == "" {
				spl_l = append(spl_l[:i], spl_l[i+1:]...)
			}
		}
		md5sum := spl_l[0]
		filename := spl_l[1]
		//log.Println(ARCH, filename, strings.Contains(ARCH, filename), spl_l, len(spl_l))
		if !strings.Contains(filename, ARCH) {
			continue
		}
		//log.Println(md5sum, md5sum_local)
		if md5sum != md5sum_local {

			return md5sum, true
		} else {
			return "", false
		}
		//log.Println(md5sum,filename)
	}
	return "", false
}

func check_and_update() {
	md5sum, need_update := check_version()
	//log.Println(md5sum)
	if need_update {
		log.Printf("we need update to %s,new client md5sum: %s", last_release.TagName, md5sum)
		down_new_client(md5sum)
	} else {
		log.Println("don't need update")
	}
}

func down_new_client(md5sum string) {
	down_path := "/tmp/bouns_manger"
	err := DownloadFile(fmt.Sprintf(GetClient, last_release.TagName, ARCH), down_path)
	if err != nil {
		log.Printf("Download new client fail: %s", err)
	}
	if md5sum == "" {
		err = CopyfileForce(os.Args[0], down_path)
		if err != nil {
			log.Printf("Copy to %s failed:%s", os.Args[0], err)
		}
	} else {
		down_ed_md5 := Getfilemd5(down_path)
		if down_ed_md5 != md5sum {
			log.Printf("down load file md5sum:%s not equal get file md5sum:%s", down_ed_md5, md5sum)
			return
		} else {
			err = CopyfileForce(os.Args[0], down_path)
		}
	}
	if err := os.Chmod(os.Args[0], 0755); err != nil {
		log.Printf("chmod failed %s", down_path)
		return
	}
	log.Println("restarting server...,if this not restart, you should be run :\nsystemctl start bonus_manger")
	os.Exit(1)
}

func check_node() error {
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
	DownloadFile(fmt.Sprintf(Bxc_node_URL, ARCH), "/opt/bcloud/nodeapi/node")
	err = os.Chmod("/opt/bcloud/nodeapi/node", 0755)
	if err != nil {
		return err
	}
	_, err = os.Create("/opt/bcloud/node.db")
	if err != nil {
		return err
	}
	ioutil.WriteFile("/lib/systemd/system/bxc-node.service", []byte(Bxc_node_service), 0644)
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

func CopyfileForce(dstName, srcName string) error {
	//log.Println("cp", "-f", srcName, dstName)
	//cmd := exec.Command("cp", "-f", srcName, dstName)
	//err := cmd.Start()
	//if err != nil {
	//	return err
	//}

	return tools.RunCommand(fmt.Sprintf("cp -f %s %s", srcName, dstName))
	//return nil
}
func Set_arch() {
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
	resp, err := http.Get(Github_latest)
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
	last_release = release
	return release, nil

}
