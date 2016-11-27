package kr

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

//	Find home directory of logged-in user even when run as sudo
func UnsudoedHomeDir() (home string) {
	userName := os.Getenv("SUDO_USER")
	if userName == "" {
		userName = os.Getenv("USER")
	}
	user, err := user.Lookup(userName)
	if err == nil && user != nil {
		home = user.HomeDir
	} else {
		log.Notice("falling back to $HOME")
		home = os.Getenv("HOME")
		err = nil
	}
	return
}

func KrDir() (krPath string, err error) {
	home := UnsudoedHomeDir()
	if err != nil {
		return
	}
	krPath = filepath.Join(home, ".kr")
	err = os.MkdirAll(krPath, os.FileMode(0700))
	return
}

func KrDirFile(file string) (fullPath string, err error) {
	krPath, err := KrDir()
	if err != nil {
		return
	}
	fullPath = filepath.Join(krPath, file)
	return
}

const DAEMON_SOCKET_FILENAME = "krd.sock"

func DaemonListen() (listener net.Listener, err error) {
	socketPath, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	//	delete UNIX socket in case daemon was not killed cleanly
	_ = os.Remove(socketPath)
	listener, err = net.Listen("unix", socketPath)
	return
}

func pingDaemon() (err error) {
	conn, err := DaemonDial()
	if err != nil {
		return
	}

	pingRequest, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		return
	}
	err = pingRequest.Write(conn)
	if err != nil {
		return
	}
	responseReader := bufio.NewReader(conn)
	_, err = http.ReadResponse(responseReader, pingRequest)
	if err != nil {
		err = fmt.Errorf("Daemon Read error: %s", err.Error())
		return
	}
	return
}

func DaemonDialWithTimeout() (conn net.Conn, err error) {
	done := make(chan error, 1)
	go func() {
		done <- pingDaemon()
	}()

	select {
	case <-time.After(time.Second):
		err = fmt.Errorf("ping timed out")
		return
	case err = <-done:
	}
	if err != nil {
		return
	}

	conn, err = DaemonDial()
	return
}