package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"
)

func Test_Getfilemd5(t *testing.T) {
	//noinspection SpellCheckingInspection
	hash := Getfilemd5(os.Args[0])
	t.Log(hash)
	fmt.Println(runtime.GOARCH)
}

func Test_check_and_update(t *testing.T)  {
	checkAndUpdate()
}
func TestDownloadFileWget(t *testing.T) {
	DownloadFileWget("https://github.com/qinghon/BonusManger/releases/download/v0.3.12/bonus_manger_aarch64","/tmp/bonus_manager")
}
func TestCopyForce(t *testing.T) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	t.Log(os.Args[0])
	t.Log(Getfilemd5(os.Args[0]), Getfilemd5("build/bonus_manger_x86_64"))
	err:= CopyForce(os.Args[0],"build/bonus_manger_x86_64")
	t.Log(Getfilemd5(os.Args[0]), Getfilemd5("build/bonus_manger_x86_64"))

	if err != nil {
		t.Error(err)
	}
}
func TestGetResp(t *testing.T) {
	resp, err := GetResp(VersionURLS2, "latest")
	if err != nil {
		t.Error(err)
	}
	t.Log(string(resp))
}
func TestGetResp2(t *testing.T) {
	resp, err := GetResp(VersionURLS, "v0.4.0")
	if err != nil {
		t.Error(err)
	}
	t.Log(string(resp))
}