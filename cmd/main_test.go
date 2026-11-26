package main

import (
	"bytes"
	"crypto/rand"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/javyliu/aes_channel/internal"
	"github.com/javyliu/aes_channel/pkg/aescrypto"
)

// TestEndToEndPipe 采用 net.Pipe 在内存中全双工模拟：
// Client <-> EncryptProxy <-> DecryptProxy <-> TargetServer
func TestEndToEndPipe(t *testing.T) {
	key := "my-secret-key"
	cipher, err := aescrypto.New(key)
	if err != nil {
		t.Fatalf("failed to init cipher: %v", err)
	}

	// 1. Target Echo 服务与 Decrypt 端连接
	targetServerConn, targetClientConn := net.Pipe()
	go func() {
		defer targetServerConn.Close()
		_, _ = io.Copy(targetServerConn, targetServerConn)
	}()

	// 2. Encrypt 端与 Decrypt 端连接
	tunnelEncConn, tunnelDecConn := net.Pipe()

	// 3. Client 与 Encrypt 端连接
	clientSideConn, proxySideConn := net.Pipe()

	runRelay := func(c1, c2 net.Conn, mode int) {
		cA := internal.NewClient(c1, 0)
		cB := internal.NewClient(c2, 0)

		var wg sync.WaitGroup
		var once sync.Once
		closeBoth := func() {
			_ = c1.Close()
			_ = c2.Close()
		}
		wg.Add(2)

		go func() {
			defer wg.Done()
			if err := cipher.ReadAndWriteStream(*cA, *cB, mode); err != nil {
				once.Do(closeBoth)
				return
			}
			_ = c2.Close()
		}()

		go func() {
			defer wg.Done()
			if err := cipher.ReadAndWriteStream(*cB, *cA, revertMode(mode)); err != nil {
				once.Do(closeBoth)
				return
			}
			_ = c1.Close()
		}()

		wg.Wait()
	}

	// 启动 Decrypt Proxy (Mode 2): tunnelDecConn <-> targetClientConn
	go runRelay(tunnelDecConn, targetClientConn, aescrypto.Decrypt)

	// 启动 Encrypt Proxy (Mode 1): proxySideConn <-> tunnelEncConn
	go runRelay(proxySideConn, tunnelEncConn, aescrypto.Encrypt)

	// 客户端收发测试
	testPayload := make([]byte, 64*1024) // 64KB
	_, _ = io.ReadFull(rand.Reader, testPayload)

	done := make(chan struct{})
	recvBuf := make([]byte, len(testPayload))

	go func() {
		defer close(done)
		_, err := io.ReadFull(clientSideConn, recvBuf)
		if err != nil {
			t.Errorf("read full error: %v", err)
		}
	}()

	_, err = clientSideConn.Write(testPayload)
	if err != nil {
		t.Fatalf("write error: %v", err)
	}

	select {
	case <-done:
		if !bytes.Equal(testPayload, recvBuf) {
			t.Fatal("echoed data does not match sent data")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for echo response")
	}
	_ = clientSideConn.Close()
}
