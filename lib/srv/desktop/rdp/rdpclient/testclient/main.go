//+build desktop_access_beta

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/websocket"

	"github.com/gravitational/teleport/lib/srv/desktop/deskproto"
	"github.com/gravitational/teleport/lib/srv/desktop/rdp/rdpclient"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("usage: %s host:port user password", os.Args[0])
	}
	addr := os.Args[1]
	username := os.Args[2]
	password := os.Args[3]

	assetPath := filepath.Join(exeDir(), "testclient")
	log.Printf("serving assets from %q", assetPath)
	http.Handle("/", http.FileServer(http.Dir(assetPath)))
	http.Handle("/connect", handleConnect(addr, username, password))

	log.Println("listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe failed:", err)
	}
}

func handleConnect(addr, username, password string) http.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		c, err := rdpclient.New(rdpclient.Options{
			Addr:         addr,
			Username:     username,
			Password:     password,
			ClientWidth:  800,
			ClientHeight: 800,
			OutputMessage: func(m deskproto.Message) error {
				data, err := m.Encode()
				if err != nil {
					return fmt.Errorf("failed to encode output message: %w", err)
				}
				return websocket.Message.Send(conn, data)
			},
			InputMessage: func() (deskproto.Message, error) {
				var buf []byte
				if err := websocket.Message.Receive(conn, &buf); err != nil {
					return nil, fmt.Errorf("failed to read input message: %w", err)
				}
				return deskproto.Decode(buf)
			},
		})
		if err != nil {
			log.Fatalf("failed to create rdpclient: %v", err)
		}
		if err := c.Wait(); err != nil {
			log.Fatalf("failed to wait for rdpclient to finish: %v", err)
		}
	})
}

func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Println("failed to find executable path:", err)
		return "."
	}
	return filepath.Dir(exe)
}