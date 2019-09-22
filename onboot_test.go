package main

import (
	"fmt"
	"runtime"
	"testing"
)

func Test_Getfilemd5(t *testing.T) {
	hash := Getfilemd5("/home/wusheng/Github/BonusManger/build/x86_64/bonus_manger")
	t.Log(hash)
	fmt.Println(runtime.GOARCH)
}

func Test_check_and_update(t *testing.T)  {
	check_and_update()
}