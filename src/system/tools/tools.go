package tools

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	user2 "os/user"
	"path/filepath"
)

type Key struct {
	PublicKey  string `json:"public_key"`  //sha openssl public key
	PrivateKey string `json:"private_key"` // sha openssl private key
}

func RunCommand(cmd_str string) error {
	cmd := exec.Command("sh", "-c", cmd_str)
	log.Printf("sh -c %s", cmd_str)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

/*获取当前文件执行的路径*/
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])

	//得到全路径，比如在windows下E:\\golang\\test\\a.exe
	path, _ := filepath.Abs(file)

	rst := filepath.Dir(path)

	return rst
}

func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func Shutdown() error {
	return RunCommand("shutdown now")
}
func Reboot() error {
	return RunCommand("reboot")
}

func (k *Key) GenKey(bit int) error {
	// private key generate
	pri, err := rsa.GenerateKey(rand.Reader, bit)
	if err != nil {
		return err
	}
	derStream := x509.MarshalPKCS1PrivateKey(pri)
	blk := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	pri_b := pem.EncodeToMemory(blk)
	k.PrivateKey = string(pri_b)

	// public key generate
	pub := &pri.PublicKey
	sshpub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return err
	}
	sshpub_b := ssh.MarshalAuthorizedKey(sshpub)
	/*
		// gen pubkey format pki pem
		// but I only ssh public key
		derPkix,err:=x509.MarshalPKIXPublicKey(pub)
		if err!=nil {
			return err
		}
		blk=&pem.Block{
			Type:"PUBLIC KEY",
			Bytes:derPkix,
		}
		pub_b:=pem.EncodeToMemory(blk)*/
	k.PublicKey = string(sshpub_b)
	return nil
}

//信任
func (k *Key) Trust() error {
	u, err := user2.Current()
	if err != nil {
		log.Println("not get user")
		return err
	}
	err = os.MkdirAll(u.HomeDir+"/.ssh", 700)
	if err != nil {
		log.Println("mkdir %s error:%s", u.HomeDir+"/.ssh", err)
	}
	if len(k.PrivateKey) != 0 {
		err = ioutil.WriteFile(u.HomeDir+"/.ssh/id_rsa.bm", []byte(k.PrivateKey), 600)
	}
	err = ioutil.WriteFile(u.HomeDir+"/.ssh/id_rsa.bm.pub", []byte(k.PublicKey), 644)
	if err != nil {
		log.Println("write key error:%s", err)
		return err
	}
	f, err := os.OpenFile(u.HomeDir+"/.ssh/authorized_keys", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 600)
	if err != nil {
		log.Println("write authorized_keys error:%s", err)
		return err
	}
	_, err = f.WriteString("\n" + k.PublicKey)
	if err != nil {
		log.Println("write authorized_keys error:%s", err)
		return err
	}
	return nil
}

func (k *Key) StartSsh() ([]byte, error) {
	cmd := exec.Command("sh", "-c", "systemctl enable ssh && systemctl start ssh")
	by, err := cmd.Output()
	if err == nil {
		return by, nil
	}
	cmd = exec.Command("sh", "-c", "service enable ssh&&service start ssh")
	by, err = cmd.Output()
	if err == nil {
		return by, nil
	}
	return by, err
}
