package tools

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	user2 "os/user"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Key struct {
	PublicKey  string `json:"public_key"`  //sha openssl public key
	PrivateKey string `json:"private_key"` // sha openssl private key
}

// write data to WebSocket
// the data comes from ssh server.
type wsBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (w *wsBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// connect to ssh server using ssh session.
type SshConn struct {
	// calling Write() to write data into ssh server
	StdinPipe io.WriteCloser
	// Write() be called to receive data from ssh server
	ComboOutput *wsBufferWriter
	Session     *ssh.Session
}

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"
)

type wsMsg struct {
	Type string `json:"type"`
	Cmd  []byte `json:"cmd"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
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
		log.Printf("mkdir %s error:%s", u.HomeDir+"/.ssh", err)
	}
	if len(k.PrivateKey) != 0 {
		err = ioutil.WriteFile(u.HomeDir+"/.ssh/id_rsa.bm", []byte(k.PrivateKey), 600)
	}
	err = ioutil.WriteFile(u.HomeDir+"/.ssh/id_rsa.bm.pub", []byte(k.PublicKey), 644)
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
func NewSshClient(user string, key Key) (*ssh.Client, error) {
	var err error
	var addr string
	addr = "[::1]:22"
	singer, err := ssh.ParsePrivateKey([]byte(key.PrivateKey))
	if err != nil {
		return nil, err
	}
	Config := &ssh.ClientConfig{HostKeyCallback: ssh.InsecureIgnoreHostKey()}
	Config.User = user
	Config.Auth = []ssh.AuthMethod{ssh.PublicKeys(singer)}

	//fmt.Println(addr)
	Client, err := ssh.Dial("tcp", addr, Config)
	return Client, err
}

// setup ssh shell session
// set Session and StdinPipe here,
// and the Session.Stdout and Session.Sdterr are also set.
func NewSshConn(cols, rows int, sshClient *ssh.Client) (*SshConn, error) {
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}

	// we set stdin, then we can write data to ssh server via this stdin.
	// but, as for reading data from ssh server, we can set Session.Stdout and Session.Stderr
	// to receive data from ssh server, and write back to somewhere.
	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}

	comboWriter := new(wsBufferWriter)
	//ssh.stdout and stderr will write output into comboWriter
	sshSession.Stdout = comboWriter
	sshSession.Stderr = comboWriter

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		return nil, err
	}
	// Start remote shell
	if err := sshSession.Shell(); err != nil {
		return nil, err
	}
	return &SshConn{StdinPipe: stdinP, ComboOutput: comboWriter, Session: sshSession}, nil
}

func (s *SshConn) Close() {
	if s.Session != nil {
		s.Session.Close()
	}

}

//flushComboOutput flush ssh.session combine output into websocket response
func flushComboOutput(w *wsBufferWriter, wsConn *websocket.Conn) error {
	if w.buffer.Len() != 0 {
		err := wsConn.WriteMessage(websocket.TextMessage, w.buffer.Bytes())
		if err != nil {
			return err
		}
		w.buffer.Reset()
	}
	return nil
}

//ReceiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
func (ssConn *SshConn) ReceiveWsMsg(wsConn *websocket.Conn, logBuff *bytes.Buffer, exitCh chan bool) {
	//tells other go routine quit
	defer setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			//read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				log.Println("reading webSocket message failed")
				return
			}
			//unmashal bytes into struct
			msgObj := wsMsg{wsMsgCmd, wsData, 0, 0}
			/*if err := json.Unmarshal(wsData, &msgObj); err != nil {
				logrus.WithError(err).WithField("wsData", string(wsData)).Error("unmarshal websocket message failed")
			}*/

			switch msgObj.Type {
			case wsMsgResize:
				//handle xterm.js size change
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := ssConn.Session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						log.Println("ssh pty change windows size failed")
					}
				}
			case wsMsgCmd:
				//handle xterm.js stdin
				//decodeBytes, err := base64.StdEncoding.DecodeString(msgObj.Cmd)
				decodeBytes := msgObj.Cmd
				if err != nil {
					log.Println("websock cmd string base64 decoding failed")
				}
				if _, err := ssConn.StdinPipe.Write(decodeBytes); err != nil {
					log.Println("ws cmd bytes write to ssh.stdin pipe failed")
					setQuit(exitCh)
				}
				//write input cmd to log buffer
				if _, err := logBuff.Write(decodeBytes); err != nil {
					log.Println("write received cmd into log buffer failed")
				}
			}
		}
	}
}
func (ssConn *SshConn) SendComboOutput(wsConn *websocket.Conn, exitCh chan bool) {
	//tells other go routine quit
	defer setQuit(exitCh)

	//every 120ms write combine output bytes into websocket response
	tick := time.NewTicker(time.Millisecond * time.Duration(12))
	tickPing := time.NewTicker(time.Second * 60)
	//for range time.Tick(120 * time.Millisecond){}
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			//write combine output bytes into websocket response
			if err := flushComboOutput(ssConn.ComboOutput, wsConn); err != nil {
				log.Println("ssh sending combo output to webSocket failed")
				return
			}
		case <-tickPing.C:
			wsConn.WriteMessage(websocket.PingMessage, []byte(""))
		case <-exitCh:
			return
		}
	}
}

func (ssConn *SshConn) SessionWait(quitChan chan bool) {
	if err := ssConn.Session.Wait(); err != nil {
		log.Println("ssh session wait failed")
		setQuit(quitChan)
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
