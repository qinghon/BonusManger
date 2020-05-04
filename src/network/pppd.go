package network

import (
	"bytes"
	"os/exec"
	"strings"
)

func GetPPPDVersion() (string, error) {
	cmd := exec.Command("pppd", "-h")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	version := strings.Fields(strings.Split(stderr.String(), "\n")[0])[2]
	return version, nil
}
