package embedded

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.yorun.ai/vine/util/vnet"
	"go.yorun.ai/vine/util/vpre"
)

const randomLocalRedisListen = "127.0.0.1:0"

func waitRedisReady(listenAddr string, timeout time.Duration) error {
	targetAddr, err := redisDialAddr(listenAddr)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	for {
		err = redisHello(targetAddr, 100*time.Millisecond)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wait redis ready timeout: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func redisHello(targetAddr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", targetAddr, timeout)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte("*2\r\n$5\r\nHELLO\r\n$1\r\n2\r\n"))
	if err != nil {
		return err
	}

	_, err = bufio.NewReader(conn).ReadByte()
	return err
}

func redisDialAddr(listenAddr string) (string, error) {
	host, portText, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "", err
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, portText), nil
}

func splitListen(listenAddr string) (string, int) {
	host, portText, err := net.SplitHostPort(listenAddr)
	vpre.CheckNilError(err, "parse redis listen addr failed")
	port, err := strconv.Atoi(portText)
	vpre.CheckNilError(err, "parse redis listen port failed")
	if host == "" {
		host = "0.0.0.0"
	}
	return host, port
}

func redisEndpoint(listenAddr string) string {
	host, port := splitListen(listenAddr)
	if host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = vnet.DetectHostIP()
	}
	return fmt.Sprintf("redis://%s:%d", host, port)
}
