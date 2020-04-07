package _package

import (
	"errors"
	"os/exec"
)

var PACKAGEMANGER string

func CheckExec(exe string) (bool) {
	_, err := exec.LookPath(exe)
	return err == nil
}

func checkManger() {
	if CheckExec("apt") {
		PACKAGEMANGER = "apt"
	} else if CheckExec("yum") {
		PACKAGEMANGER = "yum"
	} else if CheckExec("apk") {
		PACKAGEMANGER = "apk"
	} else if CheckExec("opkg") {
		PACKAGEMANGER = "opkg"
	} else {
		PACKAGEMANGER = ""
	}
}

func Install(p string) ([]byte, error) {
	checkManger()
	switch PACKAGEMANGER {
	case "apt":
		_,err:=AptInstall(p)
		if err!=nil {
			_,_=AptUpdate()
		}
		return AptInstall(p)
	case "yum":
		return YumInstall(p)
	default:
		return nil ,errors.New("Not supported yet. ")
	}
}
