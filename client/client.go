package main

import (
	"errors"
	"fmt"
	"github.com/SpauriRosso/dotlog"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

var users = []string{}

func main() {
	conn, err := net.Dial("tcp", "localhost:3000")
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		dotlog.Error(fmt.Sprintf("%v %v:%v", err, file, line))
		return
	}
	defer conn.Close()

	// init GUI
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		dotlog.Error(fmt.Sprintf("%v %v:%v", err, file, line))
		return
	}
	defer g.Close()
	g.SetManagerFunc(layout)

	// Keybinds CTRL + C
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		_, file, line, _ := runtime.Caller(1)
		dotlog.Error(fmt.Sprintf("%v %v:%v", err, file, line))
		return
	}
	// User input
	if err := g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, sendMessage(conn, g)); err != nil {
		_, file, line, _ := runtime.Caller(1)
		dotlog.Error(fmt.Sprintf("%v %v:%v", err, file, line))
		return
	}
	go listenForMessages(conn, g)
	// Launch GUI
	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		_, file, line, _ := runtime.Caller(1)
		dotlog.Error(fmt.Sprintf("%v %v:%v", err, file, line))
		return
	}
}

// organize display
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	// Messages box
	if v, err := g.SetView("messages", 20, 0, maxX-1, maxY-3); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Messages"
		v.Autoscroll = true
		v.Wrap = true
	}

	// connected users box
	if v, err := g.SetView("users", 0, 0, 20, maxY-3); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Online Users"
		v.Wrap = true
		updateUserList(g)
	}

	// input message box
	if v, err := g.SetView("input", 0, maxY-2, maxX-1, maxY); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Enter message"
		v.Editable = true
	}
	// set focus for input zone
	g.SetCurrentView("input")
	return nil
}

// quit GUI
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func sendMessage(conn net.Conn, g *gocui.Gui) func(*gocui.Gui, *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		message := strings.TrimSpace(v.Buffer())
		if len(message) > 0 {
			conn.Write([]byte(message))
			v.Clear() // clear field of input after sending to server
			v.SetCursor(0, 0)
		}
		return nil
	}
}

func listenForMessages(conn net.Conn, g *gocui.Gui) {
	for {
		message := make([]byte, 2048)
		length, err := conn.Read(message)
		if err != nil {
			return
		}
		// display messages in corresponding box
		g.Update(func(gui *gocui.Gui) error {
			v, err := g.View("messages")
			if err != nil {
				return err
			}
			msg := string(message[:length])

			if strings.Contains(msg, "[SYSTEM]:") {
				v.FgColor = gocui.ColorRed
			} else {
				v.FgColor = gocui.ColorWhite
			}
			v.Write([]byte(time.Now().Format("2006-02-01 15:04:05") + " " + msg + "\n"))

			if strings.Contains(msg, "joined the channel") {
				parts := strings.Split(msg, " ")
				if len(parts) > 3 {
					user := parts[1]
					users = append(users, user)
					updateUserList(g)
				}
			}
			return nil
		})
	}
}

func updateUserList(g *gocui.Gui) {
	g.Update(func(gui *gocui.Gui) error {
		v, err := g.View("users")
		if err != nil {
			return err
		}
		v.Clear()

		for _, user := range users {
			fmt.Fprintln(v, user)
		}
		return nil
	})
}
