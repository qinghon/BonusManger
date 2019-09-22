package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const VersionURLS = "http://192.168.4.2:81/Github/BonusManger/build/md5sum"

const GetClient = "http://192.168.4.2:81/Github/BonusManger/build/x86_64/bonus_manger"

var ARCH string

func check_version() (string, bool) {
	resp, err := http.Get(VersionURLS)
	if err != nil {
		log.Printf("get version failed: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		println("read body failed")
	}
	spl := strings.Split(string(body), "\n")
	md5sum_local := Getfilemd5(os.Args[0])

	switch runtime.GOARCH {
	case "amd64":
		ARCH = "x86_64"
	case "arm64":
		ARCH = "aarch64"
	case "arm":
		ARCH = "armv7l"

	}
	//arch:="x86_64"
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
		if ! strings.Contains(filename, ARCH) {
			continue
		}
		if md5sum != md5sum_local {
			//log.Println(md5sum, md5sum_local)
			return md5sum, false
		} else {
			return "", true
		}
		//log.Println(md5sum,filename)
	}
	return "", false
}

func Down_new_client(md5sum string) {
	down_path := "/tmp/bouns_manger"
	err := Download_file(GetClient, down_path)
	if err != nil {
		log.Printf("Download new client fail: %s", err)
	}
	if md5sum == "" {
		err = Copyfile_force(os.Args[0], down_path)
		if err != nil {
			log.Printf("Copy to %s failed:%s", os.Args[0], err)
		}
	} else {
		down_ed_md5 := Getfilemd5(down_path)
		if down_ed_md5 != md5sum {
			log.Printf("down load file md5sum:%s not equal get file md5sum:%s", down_ed_md5, md5sum)
			return
		} else {
			err = Copyfile_force(os.Args[0], down_path)
		}
	}
	os.Exit(0)
}

func check_and_update() {
	md5sum, sure_newversion := check_version()
	log.Println(md5sum)
	if ! sure_newversion {
		log.Println("we need update")
		Down_new_client(md5sum)
	} else {
		log.Println("don't need update")
	}
}

func check_node() {

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

/*获取当前文件执行的路径*/
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])

	//得到全路径，比如在windows下E:\\golang\\test\\a.exe
	path, _ := filepath.Abs(file)

	rst := filepath.Dir(path)

	return rst
}

func Download_file(_URL, _path string) (error) {
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

func onboot() {

	timetkm := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-timetkm.C:
			go check_and_update()
		}
	}
}

func Copyfile_force(dstName, srcName string) (error) {
	log.Println("cp", "-f", srcName, dstName)
	cmd := exec.Command("cp", "-f", srcName, dstName)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}
