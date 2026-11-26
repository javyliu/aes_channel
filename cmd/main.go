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

var (
	localIp      *string
	serverIp     *string
	key          *string
	timeout      *int
	handleMode   *int
	verbose      *bool
	globalCipher *aescrypto.AesCipher
)

// 监听连接的数据的处理模式，加密模式，复制模式，解密模式
// 加密端与解密端需互换端口
// 注： 端口的设置都以 ":" 开头，例如 ":18305"

func main() {
	localIp = flag.String("lip", tools.GetenvOrDefault("LOCAL_IP", ":18305", tools.StringParse), "本地服务监听地址,注 端口的设置都以 ':' 开头，例如 :18305")
	serverIp = flag.String("rip", tools.GetenvOrDefault("SERVER_IP", ":18304", tools.StringParse), "远程服务监听地址")
	key = flag.String("key", tools.GetenvOrDefault("AES_KEY", "test", tools.StringParse), "aes加密key")
	timeout = flag.Int("td", tools.GetenvOrDefault("TIMEOUT", 60, tools.IntParse), "连接到远程服务器的超时时间单位 秒")
	handleMode = flag.Int("mode", tools.GetenvOrDefault("AES_MODE", aescrypto.Encrypt, tools.IntParse), "监听到连接的数据的处理模式，1：加密模式，2：解密模式，3：复制模式，默认为加密模式")
	webPort := flag.String("web_port", tools.GetenvOrDefault("WEB_PORT", "", tools.StringParse), "web端口，默认不开启,主要用在本mode 为Encrypt时, 用于在移动提供一个pac文件供移动设备中使用")
	verbose = flag.Bool("v", tools.GetenvOrDefault("VERBOSE", false, tools.BoolParse), "是否开启详细日志输出，默认关闭以减少锁竞争提升吞吐量")

	flag.Parse()
	log.SetPrefix("[aes_channel] ")

	var err error
	globalCipher, err = aescrypto.New(*key)
	if err != nil {
		log.Fatalf("初始化 AES Cipher 失败: %v", err)
	}

	ln, err := net.Listen("tcp", *localIp)
	if err != nil {
		log.Fatalf("监听本地地址失败: %v", err)
	}
	defer ln.Close()

	if *webPort != "" {
		go func() {
			http.Handle("/", http.FileServer(http.Dir("./web")))
			log.Println("web server listening on http://localhost:" + *webPort)
			if err := http.ListenAndServe(*webPort, nil); err != nil {
				log.Println("web server error:", err)
			}
		}()
	}

	log.Printf("Listening on %s, forwarding to %s, mode=%d", *localIp, *serverIp, *handleMode)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("[accept_error]", err)
			continue
		}
		if *verbose {
			log.Println("[accept]", conn.RemoteAddr())
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	dialTimeout := time.Duration(*timeout) * time.Second
	if dialTimeout <= 0 {
		dialTimeout = 10 * time.Second
	}

	bConn, err := net.DialTimeout("tcp", *serverIp, dialTimeout)
	if err != nil {
		if *verbose {
			log.Println("[error_dial]", *serverIp, err)
		}
		return
	}
	defer bConn.Close()

	clientA := internal.NewClient(conn, *timeout)
	clientB := internal.NewClient(bConn, *timeout)

	if *verbose {
		log.Printf("conn established: A=%s (%s), B=%s (%s)", clientA.Id, conn.RemoteAddr(), clientB.Id, bConn.RemoteAddr())
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var once sync.Once
	closeBoth := func() {
		_ = conn.Close()
		_ = bConn.Close()
	}

	// 方向 1: A -> B
	go func() {
		defer wg.Done()
		err := globalCipher.ReadAndWriteStream(*clientA, *clientB, *handleMode)
		if err != nil {
			once.Do(closeBoth)
			return
		}
		// A 端发送 EOF，对 B 端执行半关闭（发送 TCP FIN）
		internal.CloseWrite(bConn)
	}()

	// 方向 2: B -> A
	go func() {
		defer wg.Done()
		err := globalCipher.ReadAndWriteStream(*clientB, *clientA, revertMode(*handleMode))
		if err != nil {
			once.Do(closeBoth)
			return
		}
		// B 端发送 EOF，对 A 端执行半关闭（发送 TCP FIN）
		internal.CloseWrite(conn)
	}()

	wg.Wait()

	if *verbose {
		log.Printf("conn closed: A=%s, B=%s", clientA.Id, clientB.Id)
	}
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
