package tools

import (
	"bytes"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"sync"
	"time"
)

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
				log.Error("reading webSocket message failed")
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
						log.Error("ssh pty change windows size failed")
					}
				}
			case wsMsgCmd:
				//handle xterm.js stdin
				//decodeBytes, err := base64.StdEncoding.DecodeString(msgObj.Cmd)
				decodeBytes := msgObj.Cmd
				if err != nil {
					log.Error("websock cmd string base64 decoding failed")
				}
				if _, err := ssConn.StdinPipe.Write(decodeBytes); err != nil {
					log.Error("ws cmd bytes write to ssh.stdin pipe failed")
					setQuit(exitCh)
				}
				//write input cmd to log buffer
				if _, err := logBuff.Write(decodeBytes); err != nil {
					log.Error("write received cmd into log buffer failed")
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
				log.Error("ssh sending combo output to webSocket failed")
				return
			}
		case <-tickPing.C:
			if err := wsConn.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
				log.Error("ssh write ping message error: %s", err)
			}
		case <-exitCh:
			return
		}
	}
}

func (ssConn *SshConn) SessionWait(quitChan chan bool) {
	if err := ssConn.Session.Wait(); err != nil {
		log.Error("ssh session wait failed")
		setQuit(quitChan)
	}
}

func setQuit(ch chan bool) {
	ch <- true
}