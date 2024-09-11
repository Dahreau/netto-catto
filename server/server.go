package main

import (
	"fmt"
	"github.com/SpauriRosso/dotlog"
	"net"
	"runtime"
	"time"
)

type Server struct {
	Addr    string
	Listnr  net.Listener
	QuitChn chan struct{}
	MsgChan chan []byte
}

func main() {
	serv := NewServer(":3000")
	go func() {
		for msg := range serv.MsgChan {
			res := fmt.Sprintf("[%v] [user] : %v", time.Now(), string(msg))
			dotlog.Info(res)
		}
	}()
	serv.StartServer()
}

func NewServer(Addr string) *Server {
	return &Server{
		Addr:    Addr,
		QuitChn: make(chan struct{}),
		MsgChan: make(chan []byte, 10),
	}
}

func (s *Server) StartServer() {
	listnr, err := net.Listen("tcp", s.Addr)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		txtErr := fmt.Sprintf("%v %v:%v", err, file, line)
		dotlog.Error(txtErr)
	}
	defer listnr.Close()
	s.Listnr = listnr
	go s.AcceptCon()
	<-s.QuitChn
	close(s.MsgChan)
	return
}

func (s *Server) AcceptCon() {
	for {
		con, err := s.Listnr.Accept()
		if err != nil {
			_, file, line, _ := runtime.Caller(1)
			txtErr := fmt.Sprintf("%v %v:%v", err, file, line)
			dotlog.Error(txtErr)
			continue // So that we can receive con request otherwise loop will exit and no more con can be accepted
		}
		dotlog.Info("New connection from: " + fmt.Sprintf("%v", con.RemoteAddr()))
		go s.ReadCon(con)
	}
}

func (s *Server) ReadCon(con net.Conn) {
	defer con.Close()
	buf := make([]byte, 2048)
	for {
		n, err := con.Read(buf)
		if err != nil {
			_, file, line, _ := runtime.Caller(1)
			txtErr := fmt.Sprintf("%v %v:%v", err, file, line)
			dotlog.Error(txtErr)
			continue
		}
		//msg := buf[:n]
		//dotlog.Info(string(msg))
		s.MsgChan <- buf[:n]
	}
}
