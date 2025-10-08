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

func (c *AesChiper) ReadAndWriteStream(src internal.Client, dst internal.Client, encrypt bool) error {
	var stream cipher.Stream
	var iv = c.Iv
	if encrypt {
		if _, err := rand.Read(*iv); err != nil {
			return err
		}

		// 将 IV 写入目标连接，以便解密时使用
		if _, err := dst.Conn.Write(*iv); err != nil {
			return err
		}
		// stream = cipher.NewCFBEncrypter(*c.Block, iv)
	} else {
		if _, err := io.ReadFull(src.Conn, *iv); err != nil {
			return err
		}

	}
	stream = cipher.NewCTR(*c.Block, *iv)

	writer := cipher.StreamWriter{S: stream, W: dst.Conn}
	_, err := io.Copy(writer, src.Conn)
	log.Println("[io_copy]", src.Id, dst.Id)

	if err != nil {
		log.Println("[error_copy]", src.Id, dst.Id, err)
		return err
	}
	return nil
}
