package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/javyliu/aes_channel/internal"
	"github.com/javyliu/aes_channel/pkg/aescrypto"
	"github.com/javyliu/aes_channel/pkg/tools"
)

var localIp *string
var serverIp *string
var key *string
var timeout *int
var handleMode *int

// 监听连接的数据的处理模式，加密模式，复制模式，解密模式
// 加密端与解密端需互换端口
// 注： 端口的设置都以 ":" 开头，例如 ":18305

func main() {
	localIp = flag.String("lip", tools.GetenvOrDefault("LOCAL_IP", ":18305", tools.StringParse), "本地服务监听地址,注 端口的设置都以 ':' 开头，例如 :18305")
	serverIp = flag.String("rip", tools.GetenvOrDefault("SERVER_IP", ":18304", tools.StringParse), "远程服务监听地址")
	key = flag.String("key", tools.GetenvOrDefault("AES_KEY", "test", tools.StringParse), "aes加密key")
	timeout = flag.Int("td", tools.GetenvOrDefault("TIMEOUT", 60, tools.IntParse), "连接到远程服务器的超时时间单位 秒")
	handleMode = flag.Int("mode", tools.GetenvOrDefault("AES_MODE", aescrypto.Encrypt, tools.IntParse), "监听到连接的数据的处理模式，1：加密模式，2：解密模式，3：复制模式，默认为加密模式")
	webPort := flag.String("web_port", tools.GetenvOrDefault("WEB_PORT", "", tools.StringParse), "web端口，默认不开启,主要用在本mode 为Encrypt时, 用于在移动提供一个pac文件供移动设备中使用")

	flag.Parse()
	log.SetPrefix("[local] ")
	ln, err := net.Listen("tcp", *localIp)
	if err != nil {
		log.Println(err)
		return
	}

	if *webPort != "" {
		go func() {
			http.Handle("/", http.FileServer(http.Dir("./web")))
			log.Println("listen on http://localhost:", *webPort)
			if err := http.ListenAndServe(*webPort, nil); err != nil {
				log.Println(err)
			}
		}()
	}

	log.Println("listen on :", *localIp)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("[accept]", conn.RemoteAddr())
		go handleConn(conn)
	}

}

func handleConn(conn net.Conn) {
	var wg sync.WaitGroup

	defer log.Println("----conn is closed")

	defer conn.Close()
	bConn, err := net.DialTimeout("tcp", *serverIp, time.Second*5)

	// 发送到服务B
	if err != nil {
		log.Println(&bConn, "[error_dial]", err)
		return
	}
	defer bConn.Close()

	aeschiper, err := aescrypto.New(*key)
	if err != nil {
		log.Println(err)
		return
	}

	clientA := internal.NewClient(conn, *timeout)
	clientB := internal.NewClient(bConn, *timeout)

	log.Println("AconnId:", clientA.Id, ", BconnId:", clientB.Id)
	wg.Add(2)

	// 加密并发送到服务B
	go func() {
		defer wg.Done()
		defer log.Println("[-------A closed]", clientA.Id)
		aeschiper.ReadAndWriteStream(*clientA, *clientB, *handleMode)
	}()

	// 从服务B读取并解密然后发送到客户端
	go func() {
		defer wg.Done()
		defer log.Println("[-------B  closed]", clientB.Id)
		aeschiper.ReadAndWriteStream(*clientB, *clientA, revertMode(*handleMode))
	}()

	wg.Wait()

}

// 加密的反操作就是解密
// 解密的反操作就是加密
// 复制的反操作就是复制
func revertMode(mode int) int {
	switch mode {
	case aescrypto.Encrypt:
		return aescrypto.Decrypt
	case aescrypto.Decrypt:
		return aescrypto.Encrypt
	case aescrypto.Copy:
		return aescrypto.Copy
	default:
		return aescrypto.Copy
	}
}
