package aescrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"log"

	"github.com/javyliu/aes_channel/internal"
)

// func init() {
// 	log.SetPrefix("[aescbc] ")
// }

// 用于定义监听到连接的数据的处理模式，加密模式，复制模式，解密模式，默认加密
// 加密模式下，把a端接收到的数据加密后发给b端，
// 解密模式下，把a端发来的数据解密后发给b端
// 复制模式下，把a端发来的数据直接发给b端
const (
	Encrypt = iota + 1
	Decrypt
	Copy
)

type AesChiper struct {
	Block   *cipher.Block
	Iv      *[]byte
	AconnId string
	BconnId string
}

func New(key string) (*AesChiper, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)

	return &AesChiper{
		Block: &block,
		Iv:    &iv,
	}, nil
}

func (c *AesChiper) ReadAndWriteStream(src internal.Client, dst internal.Client, mode int) error {
	var stream cipher.Stream
	var iv = c.Iv

	switch mode {
	case Encrypt:
		if _, err := rand.Read(*iv); err != nil {
			return err
		}
		// 将 IV 写入目标连接，以便解密时使用
		if _, err := dst.Conn.Write(*iv); err != nil {
			return err
		}
	case Decrypt:
		// 从源连接中读取 IV,以便解密
		if _, err := io.ReadFull(src.Conn, *iv); err != nil {
			return err
		}
	case Copy:
		if _, err := io.Copy(dst.Conn, src.Conn); err != nil {
			return err
		}
		return nil
	}

	stream = cipher.NewCTR(*c.Block, *iv)

	// 创建 StreamWriter。 这是一个包装器，它将流 (stream，即 CTR 加密/解密器) 链接到底层写入器 (dst.Conn)。任何写入 writer 的数据都会先经过 CTR 处理，再写入 dst.Conn。
	writer := cipher.StreamWriter{S: stream, W: dst.Conn}
	if _, err := io.Copy(writer, src.Conn); err != nil {
		log.Println("[error_copy]", src.Id, dst.Id, err)
		return err
	}
	log.Println("[io_copy]", src.Id, dst.Id)

	return nil
}
