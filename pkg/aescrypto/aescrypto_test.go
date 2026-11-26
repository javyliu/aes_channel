package aescrypto

import (
	"bytes"
	"crypto/rand"
	"io"
	"sync"
	"testing"
)

func TestArbitraryKey(t *testing.T) {
	keys := []string{"test", "mysecret", "1234567890123456", "longkeythatisgreaterthan32bytes1234567890"}
	for _, k := range keys {
		c, err := New(k)
		if err != nil {
			t.Fatalf("New failed for key %q: %v", k, err)
		}
		if c == nil {
			t.Fatalf("expected non-nil cipher for key %q", k)
		}
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c, err := New("test")
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	raw := make([]byte, 1024*64) // 64KB
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		t.Fatalf("rand read error: %v", err)
	}

	cipherBuf := new(bytes.Buffer)
	n, err := c.Pipe(cipherBuf, bytes.NewReader(raw), Encrypt)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	if n != int64(len(raw)) {
		t.Fatalf("expected written %d, got %d", len(raw), n)
	}

	plainBuf := new(bytes.Buffer)
	n, err = c.Pipe(plainBuf, cipherBuf, Decrypt)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}
	if n != int64(len(raw)) {
		t.Fatalf("expected decrypted %d, got %d", len(raw), n)
	}

	if !bytes.Equal(plainBuf.Bytes(), raw) {
		t.Fatal("decrypted payload does not match original")
	}
}

func TestConcurrentDirectionsNoIVRace(t *testing.T) {
	// 验证双向同时加解密时不再产生 IV 数据竞争与解密损坏
	c, err := New("concurrent-test-key")
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			payload := []byte("forward stream payload data 123456")
			var buf bytes.Buffer
			_, err := c.Pipe(&buf, bytes.NewReader(payload), Encrypt)
			if err != nil {
				t.Errorf("forward encrypt err: %v", err)
			}
			var res bytes.Buffer
			_, err = c.Pipe(&res, &buf, Decrypt)
			if err != nil {
				t.Errorf("forward decrypt err: %v", err)
			}
			if !bytes.Equal(res.Bytes(), payload) {
				t.Errorf("forward mismatch")
			}
		}()

		go func() {
			defer wg.Done()
			payload := []byte("backward stream response payload 789012")
			var buf bytes.Buffer
			_, err := c.Pipe(&buf, bytes.NewReader(payload), Encrypt)
			if err != nil {
				t.Errorf("backward encrypt err: %v", err)
			}
			var res bytes.Buffer
			_, err = c.Pipe(&res, &buf, Decrypt)
			if err != nil {
				t.Errorf("backward decrypt err: %v", err)
			}
			if !bytes.Equal(res.Bytes(), payload) {
				t.Errorf("backward mismatch")
			}
		}()
	}
	wg.Wait()
}

func BenchmarkPipeEncrypt(b *testing.B) {
	c, _ := New("benchmark-key-1234")
	data := make([]byte, 32*1024) // 32KB chunk
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		src := bytes.NewReader(data)
		_, _ = c.Pipe(io.Discard, src, Encrypt)
	}
}

func BenchmarkPipeDecrypt(b *testing.B) {
	c, _ := New("benchmark-key-1234")
	data := make([]byte, 32*1024)
	var cipherBuf bytes.Buffer
	_, _ = c.Pipe(&cipherBuf, bytes.NewReader(data), Encrypt)
	cipherBytes := cipherBuf.Bytes()

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		src := bytes.NewReader(cipherBytes)
		_, _ = c.Pipe(io.Discard, src, Decrypt)
	}
}
