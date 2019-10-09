package _package

import "os/exec"

func AptInstall(p string) ([]byte,error) {
	cmd:=exec.Command("apt","install","-y",p)
	return cmd.Output()
}
func AptUpdate() ([]byte,error) {
	cmd:=exec.Command("apt","update")
	return cmd.Output()
}
