package aescrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"sync"

	"github.com/javyliu/aes_channel/internal"
)

// 数据处理模式：加密模式，解密模式，复制模式，默认为加密模式
const (
	Encrypt = iota + 1
	Decrypt
	Copy
)

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// AesCipher AES 加解密器
type AesCipher struct {
	block cipher.Block
}

// AesChiper 兼容历史拼写别名
type AesChiper = AesCipher

// deriveKey 将任意长度的密钥派生为合法的 AES 密钥 (16, 24 或 32 字节)
// 若用户传入的密钥长度已为 16/24/32 字节则直接使用，否则采用 SHA-256 派生 32 字节密钥 (AES-256)
func deriveKey(key string) []byte {
	k := []byte(key)
	switch len(k) {
	case 16, 24, 32:
		return k
	default:
		h := sha256.Sum256(k)
		return h[:]
	}
}

// New 创建一个新的 AesCipher 实例
func New(key string) (*AesCipher, error) {
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return nil, err
	}
	return &AesCipher{
		block: block,
	}, nil
}

// Pipe 从 src 读取数据，处理后写入 dst。
// 针对 Encrypt / Decrypt 模式进行原地 AES-CTR 加解密，利用 sync.Pool 消除高频堆内存分配；
// 每个流独立分配 16 字节 IV，彻底解决双向并发竞争问题。
func (c *AesCipher) Pipe(dst io.Writer, src io.Reader, mode int) (int64, error) {
	var stream cipher.Stream

	switch mode {
	case Encrypt:
		var iv [aes.BlockSize]byte
		if _, err := io.ReadFull(rand.Reader, iv[:]); err != nil {
			return 0, err
		}
		// 将 IV 写入目标连接首部
		if _, err := dst.Write(iv[:]); err != nil {
			return 0, err
		}
		stream = cipher.NewCTR(c.block, iv[:])

	case Decrypt:
		var iv [aes.BlockSize]byte
		// 从源连接首部读取 IV
		if _, err := io.ReadFull(src, iv[:]); err != nil {
			return 0, err
		}
		stream = cipher.NewCTR(c.block, iv[:])

	case Copy:
		// 复制模式，不加解密
	default:
		return 0, fmt.Errorf("invalid mode: %d", mode)
	}

	bufPtr := bufPool.Get().(*[]byte)
	buf := *bufPtr
	defer bufPool.Put(bufPtr)

	var written int64
	for {
		nr, rErr := src.Read(buf)
		if nr > 0 {
			if stream != nil {
				// 原地异或流加解密，零堆内存分配
				stream.XORKeyStream(buf[:nr], buf[:nr])
			}
			nw, wErr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if wErr != nil {
				return written, wErr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				return written, nil
			}
			return written, rErr
		}
	}
}

// ReadAndWriteStream 兼容旧版调用签名
func (c *AesCipher) ReadAndWriteStream(src internal.Client, dst internal.Client, mode int) error {
	_, err := c.Pipe(dst.Conn, src.Conn, mode)
	return err
}
