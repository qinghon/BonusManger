package tools

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	user2 "os/user"
	"path"
	"path/filepath"
)

type Key struct {
	PublicKey  string `json:"public_key"`  //sha openssl public key
	PrivateKey string `json:"private_key"` // sha openssl private key
}

// write data to WebSocket
// the data comes from ssh server.


func RunCommand(cmd_str string) error {
	cmd := exec.Command("sh", "-c", cmd_str)
	log.Printf("bash -c %s", cmd_str)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

/*获取当前文件执行的路径*/
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])

	//得到全路径，比如在windows下E:\\golang\\test\\a.exe
	_path, _ := filepath.Abs(file)

	rst := filepath.Dir(_path)

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
		log.Printf("mkdir %s error:%s", u.HomeDir+"/.ssh", err)
	}
	if len(k.PrivateKey) != 0 {
		err = ioutil.WriteFile(u.HomeDir+"/.ssh/id_rsa.bm", []byte(k.PrivateKey), 400)
	}
	err = ioutil.WriteFile(u.HomeDir+"/.ssh/id_rsa.bm.pub", []byte(k.PublicKey), 400)
	if err != nil {
		log.Printf("write key error:%s", err)
		return err
	}
	f, err := os.OpenFile(u.HomeDir+"/.ssh/authorized_keys", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 600)
	if err != nil {
		log.Printf("write authorized_keys error:%s", err)
		return err
	}
	_, err = f.WriteString("\n" + k.PublicKey)
	if err != nil {
		log.Printf("write authorized_keys error:%s", err)
		return err
	}
	return nil
}

func (k *Key) IsTrust() bool {
	u, err := user2.Current()
	if err != nil {
		log.Println("not get user")
		return false
	}
	au, err := ioutil.ReadFile(path.Join(u.HomeDir, ".ssh", "authorized_keys"))
	if err != nil {
		log.Printf("not get authorized_keys: %s", err)
		return false
	}
	if bytes.Contains(au, []byte(k.PublicKey)) {
		return true
	} else {
		return false
	}
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

//ReadHostKey
func ReadHostKey(u string) ([]Key, error) {
	var keys []Key
	userP, err := user2.Lookup(u)
	if err != nil {
		return nil, err
	}
	homedir := userP.HomeDir
	files, err := ioutil.ReadDir(filepath.Join(homedir, ".ssh"))
	for _, fi := range files {
		priContent, err := ioutil.ReadFile(filepath.Join(homedir, ".ssh", fi.Name()))
		if err != nil {
			continue
		}
		k := Key{}
		if bytes.Contains(priContent, []byte("-----BEGIN RSA PRIVATE KEY-----")) {
			k.PrivateKey = string(priContent)
			pubCotent, err := ioutil.ReadFile(filepath.Join(homedir, ".ssh", fi.Name()+".pub"))
			if err != nil {
				continue
			}
			if bytes.Contains(pubCotent, []byte("ssh-rsa")) {
				k.PublicKey = string(pubCotent)
				keys = append(keys, k)
			}
		}
	}
	return keys, nil
}

func CmdStdout(cmd *exec.Cmd) ([]byte,error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var err, errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()
	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()
	err = cmd.Wait()
	if err != nil {
		return nil,err
	}
	if errStdout != nil || errStderr != nil {
		return nil,errStderr
	}
	Out:=stdoutBuf.Bytes()
	ErrOut:=stderrBuf.Bytes()

	return append(Out,ErrOut...),err
}

func GetFileContentType(out *os.File) (string, error) {
	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	return contentType, nil
}