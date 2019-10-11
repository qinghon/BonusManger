package _package

import "os/exec"

func YumInstall(p string) ([]byte,error) {
	cmd:=exec.Command("yum","install","-y",p)
	return cmd.Output()
}
