package _package

import (
	"errors"
	"os/exec"
)

var PACKAGE_MANGER string

func CheckExec(exe string) (bool) {
	_, err := exec.LookPath(exe)
	return err == nil
}

func checkManger() {
	if CheckExec("apt") {
		PACKAGE_MANGER = "apt"
	} else if CheckExec("yum") {
		PACKAGE_MANGER = "yum"
	} else if CheckExec("apk") {
		PACKAGE_MANGER = "apk"
	} else if CheckExec("opkg") {
		PACKAGE_MANGER = "opkg"
	} else {
		PACKAGE_MANGER = ""
	}
}

func Install(p string) ([]byte, error) {
	checkManger()
	switch PACKAGE_MANGER {
	case "apt":
		_,err:=AptInstall(p)
		if err!=nil {
			_,_=AptUpdate()
		}
		return AptInstall(p)
	case "yum":
		return YumInstall(p)
	default:
		return nil ,errors.New("Not supported yet")
	}
}
