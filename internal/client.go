package internal

import (
	"fmt"
	"net"
	"sync/atomic"
)

var execCount atomic.Uint64

type Client struct {
	Conn net.Conn
	Id   string
}

func NewClient(conn net.Conn, timeout int) *Client {
	id := ConnID(conn)
	return &Client{
		Conn: conn,
		Id:   id,
	}
}

func ConnID(conn net.Conn) string {
	id := execCount.Add(1)
	return fmt.Sprintf("conid[%d]", id)
}

// CloseWriter 支持 TCP 半关闭的接口
type CloseWriter interface {
	CloseWrite() error
}

// CloseWrite 若连接支持半关闭则调用 CloseWrite 发送 TCP FIN
func CloseWrite(conn net.Conn) {
	if cw, ok := conn.(CloseWriter); ok {
		_ = cw.CloseWrite()
	}
}
