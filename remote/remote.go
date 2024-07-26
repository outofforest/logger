package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/outofforest/parallel"
	"github.com/pkg/errors"
	"github.com/ridge/must"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/outofforest/logger"
)

const (
	batchSize     = 100
	retryInterval = time.Second
)

// WithRemote adds remote logging to the logger in the context and returns a
// modified context. The logger in the returned context logs both to stderr and
// to the remote loki endpoint.
//
// The caller must call the returned cleanup function after using the logger.
func WithRemote(ctx context.Context, lokiAddr string) (context.Context, parallel.Task) {
	conn := &lokiConn{
		lokiAddr: lokiAddr,
		buffer:   make(chan interface{}, 1000),
		lastTime: time.Now().UnixNano(),
		batch:    make(chan []byte, batchSize),
		syncs:    make(chan chan struct{}, batchSize),
	}

	remoteCore := zapcore.NewCore(zapcore.NewJSONEncoder(logger.EncoderConfig), conn, zap.NewAtomicLevelAt(zap.DebugLevel))

	log := logger.Get(ctx)
	log = log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, remoteCore)
	}))

	return logger.WithLogger(ctx, log), conn.Run
}

type lokiConn struct {
	lokiAddr string
	buffer   chan interface{}
	lastTime int64
	batch    chan []byte
	syncs    chan chan struct{}
}

func (lc *lokiConn) Run(ctx context.Context) error {
	for {
		watcher := time.After(10 * time.Second)

		select {
		case i := <-lc.buffer:
			switch item := i.(type) {
			case []byte:
				lc.batch <- item
				if len(lc.batch) < cap(lc.batch) {
					continue
				}
			case chan struct{}:
				select {
				case lc.syncs <- item:
				default:
					<-lc.syncs
					lc.syncs <- item
				}
			}
		case <-watcher:
		case <-ctx.Done():
			return ctx.Err()
		}

		if err := lc.send(ctx); err != nil {
			return err
		}

	loop:
		for {
			select {
			case ch := <-lc.syncs:
				close(ch)
			default:
				break loop
			}
		}
	}
}

func (lc *lokiConn) Write(entry []byte) (int, error) {
	// zap uses reusable buffers so data have to be copied before enqueuing
	data := make([]byte, len(entry))
	copy(data, entry)

	return len(data), lc.enqueue(data)
}

func (lc *lokiConn) Sync() error {
	ch := make(chan struct{})
	if err := lc.enqueue(ch); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return errors.WithStack(ctx.Err())
	}
}

func (lc *lokiConn) enqueue(item interface{}) error {
	select {
	case lc.buffer <- item:
	default:
		return errors.New("no space in buffer")
	}
	return nil
}

type logItem struct {
	Time int64
	Data []byte
}

type logKey struct {
	Level  string
	Logger string
}

func (lc *lokiConn) send(ctx context.Context) error {
	if len(lc.batch) == 0 {
		return nil
	}

	items := map[logKey][]logItem{}
loop:
	for {
		select {
		case rawItem := <-lc.batch:
			data := map[string]any{}
			must.OK(json.Unmarshal(rawItem, &data))

			tsParsed, err := time.Parse(time.RFC3339Nano, data[logger.EncoderConfig.TimeKey].(string))
			must.OK(err)
			ts := tsParsed.UnixNano()
			if lc.lastTime > ts {
				ts = lc.lastTime
			}

			level := data[logger.EncoderConfig.LevelKey]
			name := data[logger.EncoderConfig.NameKey]
			if name == nil {
				name = ""
			}

			delete(data, logger.EncoderConfig.TimeKey)
			delete(data, logger.EncoderConfig.LevelKey)
			delete(data, logger.EncoderConfig.NameKey)

			key := logKey{
				Level:  level.(string),
				Logger: name.(string),
			}
			if items[key] == nil {
				items[key] = []logItem{}
			}

			items[key] = append(items[key], logItem{
				Time: ts,
				Data: must.Bytes(json.Marshal(data)),
			})
		default:
			break loop
		}
	}
	for _, is := range items {
		is := is
		sort.Slice(is, func(i, j int) bool {
			return is[i].Time < is[j].Time
		})

		lastTS := is[len(is)-1].Time
		if lastTS > lc.lastTime {
			lc.lastTime = lastTS
		}
	}

	streams := []any{}
	for k, is := range items {
		values := make([]any, 0, len(is))
		for _, item := range is {
			values = append(values, []any{
				strconv.FormatInt(item.Time, 10),
				string(item.Data),
			})
		}

		streams = append(streams, map[string]any{
			"stream": map[string]any{
				"level":  k.Level,
				"logger": k.Logger,
			},
			"values": values,
		})
	}

	log := logger.Get(ctx)
	for {
		err := func() error {
			reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			req := must.HTTPRequest(http.NewRequestWithContext(reqCtx, http.MethodPost, lc.lokiAddr+"/loki/api/v1/push", bytes.NewReader(must.Bytes(json.Marshal(map[string]any{"streams": streams})))))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.WithStack(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return errors.WithStack(err)
				}

				switch resp.StatusCode {
				case http.StatusBadRequest:
					log.Error("Received Bad Request response from loki", zap.ByteString("body", body))
				default:
					return errors.Errorf("unexpected response from loki endpoint, code: %d, body: %s", resp.StatusCode, body)
				}
			}

			return nil
		}()

		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		case <-time.After(retryInterval):
		}
	}
}
