package kv

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/rueidis"
)

type KV[T any] struct {
	client rueidis.Client
}

func New[T any](
	ctx context.Context,
	address string,
) (*KV[T], error) {
	opt, err := rueidis.ParseURL(address)
	if err != nil {
		opt = rueidis.ClientOption{
			InitAddress: []string{address},
		}
	}

	if strings.HasPrefix(address, "redis://") {
		opt.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS13,
		}

		opt.DisableCache = true
	}

	cli, err := rueidis.NewClient(opt)
	if err != nil {
		return nil, fmt.Errorf("failed new kv client: %w", err)
	}

	ping := cli.B().Ping().Build()

	if err := cli.Do(ctx, ping).Error(); err != nil {
		return nil, fmt.Errorf("failed to ping kv: %w", err)
	}

	return &KV[T]{
		client: cli,
	}, nil
}

func (kv *KV[T]) Set(
	ctx context.Context,
	key string,
	value T,
) error {
	val := bytes.Buffer{}

	if err := json.NewEncoder(&val).Encode(value); err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	cmd := kv.client.B().Set().Key(key).Value(rueidis.BinaryString(val.Bytes())).Build()

	if err := kv.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to set: %w", err)
	}

	return nil
}

func (kv *KV[T]) GetDel(
	ctx context.Context,
	key string,
) (T, error) {
	var value T

	cmd := kv.client.B().Getdel().Key(key).Build()

	val, err := kv.client.Do(ctx, cmd).ToString()
	if err != nil {
		return value, fmt.Errorf("failed to getdel: %w", err)
	}

	if err := json.NewDecoder(bytes.NewBufferString(val)).Decode(&value); err != nil {
		return value, fmt.Errorf("failed to decode: %w", err)
	}

	return value, nil
}
