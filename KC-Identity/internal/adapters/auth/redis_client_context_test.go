package auth

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	stalledRedisCallTimeout  = 100 * time.Millisecond
	stalledRedisCallLimit    = 2 * time.Second
	stalledRedisPoolSize     = 1
	stalledRedisKey          = "state"
	stalledRedisNetwork      = "tcp"
	stalledRedisAddress      = "127.0.0.1:0"
	stalledRedisCommandGet   = "GET"
	stalledRedisCommandPing  = "PING"
	stalledRedisCommandHello = "HELLO"
	stalledRedisReplyOK      = "+OK\r\n"
	stalledRedisReplyPong    = "+PONG\r\n"
	stalledRedisReplyUnknown = "-ERR unknown command\r\n"
	stalledRedisArrayPrefix  = "*"
)

// serveRedisUntilGet answers the connection handshake and then goes silent on GET.
func serveRedisUntilGet(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	reader := bufio.NewReader(conn)
	for {
		command, err := readRedisCommand(reader)
		if err != nil {
			return
		}
		reply := stalledRedisReplyOK
		switch command {
		case stalledRedisCommandGet:
			_, _ = io.Copy(io.Discard, reader)
			return
		case stalledRedisCommandHello:
			reply = stalledRedisReplyUnknown
		case stalledRedisCommandPing:
			reply = stalledRedisReplyPong
		}
		if _, err := conn.Write([]byte(reply)); err != nil {
			return
		}
	}
}

func readRedisCommand(reader *bufio.Reader) (string, error) {
	header, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(header, stalledRedisArrayPrefix)))
	if err != nil {
		return "", err
	}
	var command string
	for i := range count {
		if _, err := reader.ReadString('\n'); err != nil {
			return "", err
		}
		value, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if i == 0 {
			command = strings.ToUpper(strings.TrimSpace(value))
		}
	}
	return command, nil
}

// A Redis that stops answering must not hold a request past its own deadline.
func TestRedisClient_GetStopsAtTheRequestDeadline(t *testing.T) {
	listener, err := net.Listen(stalledRedisNetwork, stalledRedisAddress)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, listener.Close()) })
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go serveRedisUntilGet(conn)
		}
	}()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	ctx := testutil.TestContext()
	client, err := NewRedisClient(ctx, &config.RedisConfig{
		Host: tcpAddr.IP.String(), Port: tcpAddr.Port, PoolSize: stalledRedisPoolSize,
	}, logger.Get())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })

	callCtx, cancel := context.WithTimeout(ctx, stalledRedisCallTimeout)
	defer cancel()
	started := time.Now()
	_, err = client.Get(callCtx, stalledRedisKey)
	require.Error(t, err)
	assert.Less(t, time.Since(started), stalledRedisCallLimit)
}
