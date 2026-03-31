package idempotency

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var ErrNotFound = errors.New("idempotency key not found")

type Store interface {
	Get(ctx context.Context, key string) (string, error)
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
}

type RedisStore struct {
	addr    string
	timeout time.Duration
}

func NewRedisStore(addr string) *RedisStore {
	return &RedisStore{
		addr:    addr,
		timeout: 3 * time.Second,
	}
}

func (s *RedisStore) Get(ctx context.Context, key string) (string, error) {
	conn, br, err := s.open(ctx)
	if err != nil {
		return "", fmt.Errorf("redis connect: %w", err)
	}
	defer conn.Close()

	if err := writeArray(conn, "GET", key); err != nil {
		return "", err
	}
	typ, payload, err := readReply(br)
	if err != nil {
		return "", err
	}
	switch typ {
	case '$':
		if payload == "" {
			return "", ErrNotFound
		}
		return payload, nil
	case '-':
		return "", errors.New(payload)
	default:
		return "", fmt.Errorf("unexpected redis reply type: %q", string(typ))
	}
}

func (s *RedisStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	conn, br, err := s.open(ctx)
	if err != nil {
		return false, fmt.Errorf("redis connect: %w", err)
	}
	defer conn.Close()

	seconds := int(ttl / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	if err := writeArray(conn, "SET", key, value, "NX", "EX", strconv.Itoa(seconds)); err != nil {
		return false, err
	}
	typ, payload, err := readReply(br)
	if err != nil {
		return false, err
	}
	switch typ {
	case '+':
		return strings.EqualFold(payload, "OK"), nil
	case '$':
		// nil bulk string 表示 NX 条件未命中
		return false, nil
	case '-':
		return false, errors.New(payload)
	default:
		return false, fmt.Errorf("unexpected redis reply type: %q", string(typ))
	}
}

func (s *RedisStore) Ping(ctx context.Context) error {
	conn, br, err := s.open(ctx)
	if err != nil {
		return fmt.Errorf("redis connect: %w", err)
	}
	defer conn.Close()
	if err := writeArray(conn, "PING"); err != nil {
		return err
	}
	typ, payload, err := readReply(br)
	if err != nil {
		return err
	}
	if typ == '+' && strings.EqualFold(payload, "PONG") {
		return nil
	}
	return fmt.Errorf("unexpected ping reply: %q %q", string(typ), payload)
}

func (s *RedisStore) open(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	dialer := net.Dialer{Timeout: s.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return nil, nil, err
	}
	deadline := time.Now().Add(s.timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetDeadline(deadline)
	return conn, bufio.NewReader(conn), nil
}

func writeArray(conn net.Conn, args ...string) error {
	if _, err := fmt.Fprintf(conn, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return nil
}

func readReply(br *bufio.Reader) (byte, string, error) {
	prefix, err := br.ReadByte()
	if err != nil {
		return 0, "", err
	}
	switch prefix {
	case '+', '-', ':':
		line, err := br.ReadString('\n')
		if err != nil {
			return 0, "", err
		}
		return prefix, strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
	case '$':
		line, err := br.ReadString('\n')
		if err != nil {
			return 0, "", err
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		n, err := strconv.Atoi(line)
		if err != nil {
			return 0, "", err
		}
		if n == -1 {
			return '$', "", nil
		}
		buf := make([]byte, n+2)
		if _, err := br.Read(buf); err != nil {
			return 0, "", err
		}
		return '$', string(buf[:n]), nil
	default:
		return 0, "", fmt.Errorf("unsupported redis reply prefix: %q", string(prefix))
	}
}
