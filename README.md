# AES Channel

一个基于 AES 加密的 TCP 通道转发工具，支持加密、解密和复制三种数据处理模式。


## 功能说明


  主程序入口，监听本地端口，转发数据到远程服务，并根据模式进行加密、解密或复制。

  AES 加密/解密流处理，支持三种模式：加密、解密、复制。

## 编译
```bash
go build -o aes_channel cmd/main.go
```
## 使用方法

### 启动服务

```sh
aes_channel -lip :18305 -rip 127.0.0.1:18304 -key your_aes_key -td 60 -mode 1
```

参数说明：

- `-lip`：本地服务监听地址（默认 `:18305`）
- `-rip`：远程服务地址（默认 `:18304`）
- `-key`：AES 加密密钥（默认 `test`，建议自定义）
- `-td`：连接超时时间（秒，默认 `60`）
- `-mode`：数据处理模式  
  - `1`：加密模式（Encrypt）  
  - `2`：解密模式（Decrypt）  
  - `3`：复制模式（Copy）

**加密端与解密端需互换端口和模式。**

### 示例

- 本地加密转发：

  ```sh
  aes_channel -lip :18305 -rip 远程IP:18304 -key mysecret -mode 1
  ```

- 远程解密转发：

  ```sh
  aes_channel -lip :18304 -rip 目标服务IP:端口 -key mysecret -mode 2
  ```

### 复制模式

无需加密/解密，仅做数据转发：

```sh
aes_channel -lip :18305 -rip 远程IP:18304 -mode 3
```

## 依赖

- Go 1.23+
- 标准库：`crypto/aes`, `crypto/cipher`, `net`, `io`, `log`

## Dcoker 启动
```bash
docker run --rm -e LOCAL_IP=:18305 -e SERVER_IP=x.x.x.x:18304 -e AES_KEY=test -e TIMEOUT=60 -e AES_MODE=1 -p 18304:18302 javyliu/aes_channel
```

## 同时启动一个web服务

为了在移动端使用`自动配置代理`，在环境变量设置`WEB_PORT=:xx`, 或 在启动参数中加 `-web_port :xx`， 那么同时会启动一个静态文件访问的web服务器，提供的文件需放在在当前启动目录的web文件夹中


## 许可证

MIT License

---

详细实现请参考：[cmd/main.go](cmd/main.go)、[`internal.Client`](internal/client.go)、[`pkg/aescrypto.AesChiper`](pkg/aescrypto/aescrypto.go)

