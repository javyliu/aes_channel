package internal

import (
	"fmt"
	"net"
	"time"
)

var execCount = 0

type Client struct {
	Conn net.Conn
	Id   string
}

func NewClient(conn net.Conn, timeout int) *Client {
	id := ConnID(conn)
	if timeout > 0 {
		conn.SetDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
	}

	return &Client{
		Conn: conn,
		Id:   id,
	}
}

func ConnID(conn net.Conn) string {
	// host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	// if err != nil {
	// 	return ""
	// }
	if execCount > 100000 {
		execCount = 0
	}
	execCount++
	// return fmt.Sprintf("%v:%v", conn.RemoteAddr(), execCount)
	return fmt.Sprintf("conid[%v]", execCount)
}
