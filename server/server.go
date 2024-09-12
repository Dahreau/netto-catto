package main

import (
	"fmt"
	"github.com/SpauriRosso/dotlog"
	"net"
	"runtime"
	"sync"
)

type Client struct {
	Conn     net.Conn
	Username string
}

type Server struct {
	Addr    string
	Listnr  net.Listener
	QuitChn chan struct{}
	MsgChan chan []byte
	Clients map[net.Conn]*Client
	Mutex   sync.Mutex // Mutex synchronize access to ressources
}

func main() {
	serv := NewServer(":3000")
	go func() {
		for msg := range serv.MsgChan {
			res := fmt.Sprintf(string(msg))
			dotlog.Info(res)
			serv.Broadcast(msg)
		}
	}()
	serv.StartServer()
}

func NewServer(Addr string) *Server {
	return &Server{
		Addr:    Addr,
		QuitChn: make(chan struct{}),
		MsgChan: make(chan []byte, 10),
		Clients: make(map[net.Conn]*Client), // Initialiser la liste des clients
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
			continue
		}
		dotlog.Info("New connection from: " + fmt.Sprintf("%v", con.RemoteAddr()))
		go s.AuthenticateClient(con)
	}
}

func (s *Server) AuthenticateClient(con net.Conn) {
	buf := make([]byte, 2048)
	n, err := con.Read(buf)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		txtErr := fmt.Sprintf("%v %v:%v", err, file, line)
		dotlog.Error(txtErr)
		con.Close()
		return
	}
	username := string(buf[:n])

	// check usrname availability
	s.Mutex.Lock()
	for _, client := range s.Clients {
		if client.Username == username {
			con.Write([]byte("ERR: Please select a valid username\n"))
			con.Close()
			s.Mutex.Unlock()
			return
		}
	}
	s.Mutex.Unlock()
	dotlog.Info(fmt.Sprintf("User '%v' joined the channel", username))
	systmMsg := fmt.Sprintf("[SYSTEM]: %v joined the channel", username)
	go s.Broadcast([]byte(systmMsg))
	//con.Write([]byte(systmMsg))

	client := &Client{
		Conn:     con,
		Username: username,
	}
	s.Mutex.Lock()
	s.Clients[con] = client
	s.Mutex.Unlock()

	go s.ReadCon(client)
}

func (s *Server) ReadCon(client *Client) {
	defer func() {
		s.Mutex.Lock()
		delete(s.Clients, client.Conn)
		s.Mutex.Unlock()
		client.Conn.Close()
	}()

	buf := make([]byte, 2048)
	for {
		n, err := client.Conn.Read(buf)
		if err != nil {
			_, file, line, _ := runtime.Caller(1)
			txtErr := fmt.Sprintf("%v %v:%v", err, file, line)
			dotlog.Error(txtErr)
			return
		}
		msg := fmt.Sprintf("[%v]: %v", client.Username, string(buf[:n]))
		s.MsgChan <- []byte(msg)
	}
}

func (s *Server) Broadcast(msg []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for _, client := range s.Clients {
		if string(msg[:len(client.Username)]) == "["+client.Username {
			continue
		}
		_, err := client.Conn.Write(msg)
		if err != nil {
			dotlog.Error(fmt.Sprintf("Failed to send message to %v: %v", client.Conn.RemoteAddr(), err))
			client.Conn.Close()
			delete(s.Clients, client.Conn)
		}
	}
}
