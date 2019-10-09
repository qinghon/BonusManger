package _package

import (
	"errors"
	"os/exec"
)

var PACKAGE_MANGER string

func Check_exec(exe string) (bool) {
	_, err := exec.LookPath(exe)
	return err == nil
}

func Check_manger() {
	if Check_exec("apt") {
		PACKAGE_MANGER = "apt"
	} else if Check_exec("yum") {
		PACKAGE_MANGER = "yum"
	} else if Check_exec("apk") {
		PACKAGE_MANGER = "apk"
	} else if Check_exec("opkg") {
		PACKAGE_MANGER = "opkg"
	} else {
		PACKAGE_MANGER = ""
	}
}

func Install(p string) ([]byte, error) {
	Check_manger()
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
