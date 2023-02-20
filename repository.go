package main

import (
	"context"

	"github.com/vahid-sohrabloo/chconn/v2/chpool"
	"github.com/vahid-sohrabloo/chconn/v2/column"
	"github.com/vahid-sohrabloo/chconn/v2/types"
	"go.uber.org/zap"
)

type repository struct {
	conn chpool.Pool

	logger *zap.Logger
}

func newRepository(pool chpool.Pool, logger *zap.Logger) *repository {
	return &repository{
		conn:   pool,
		logger: logger,
	}
}

func (r *repository) ping(ctx context.Context) error {
	return r.conn.Ping(ctx)
}

// initialize must be called once before any other func invocations. It applies migrations like creation of a table
func (r *repository) initialize(ctx context.Context) error {
	err := r.conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS analytics (
		client_time DATETIME,
		device_id VARCHAR(255),
		device_os VARCHAR(255),
		session VARCHAR(255),
		sequence Int64,
		event VARCHAR(255),
		param_int Int64,
		param_str LONGTEXT,
		ip VARCHAR(255),
		server_time DATETIME
	) Engine=MergeTree() ORDER BY server_time`)

	return err
}

func (r *repository) save(ctx context.Context, batch []eventEntry) error {
	batchLen := len(batch)
	if batchLen == 0 {
		return nil
	}

	r.logger.Debug("Attempt so save batch", zap.Int("batch_len", batchLen))

	clientTimeCol := column.NewDate[types.DateTime]()
	clientTimeCol.SetWriteBufferSize(batchLen)
	deviceIdCol := column.NewString()
	deviceIdCol.SetWriteBufferSize(batchLen)
	deviceOsCol := column.NewString()
	deviceOsCol.SetWriteBufferSize(batchLen)
	sessionCol := column.NewString()
	sessionCol.SetWriteBufferSize(batchLen)
	sequenceCol := column.New[int]()
	sequenceCol.SetWriteBufferSize(batchLen)
	eventCol := column.NewString()
	eventCol.SetWriteBufferSize(batchLen)
	paramIntCol := column.New[int]()
	paramIntCol.SetWriteBufferSize(batchLen)
	paramStrCol := column.NewString()
	paramStrCol.SetWriteBufferSize(batchLen)
	ipCol := column.NewString()
	ipCol.SetWriteBufferSize(batchLen)
	serverTimeCol := column.NewDate[types.DateTime]()
	serverTimeCol.SetWriteBufferSize(batchLen)

	for _, event := range batch {
		clientTimeCol.Append(event.ClientTime.ToTime())
		deviceIdCol.Append(*event.DeviceId)
		deviceOsCol.Append(*event.DeviceOs)
		sessionCol.Append(*event.Session)
		sequenceCol.Append(*event.Sequence)
		eventCol.Append(*event.Event)
		paramIntCol.Append(*event.ParamInt)
		paramStrCol.Append(*event.ParamStr)
		ipCol.Append(event.ip)
		serverTimeCol.Append(event.serverTime)
	}

	err := r.conn.Insert(ctx, `INSERT INTO analytics (
        client_time, device_id, device_os, session, sequence, 
        event, param_int, param_str, ip, server_time
    ) VALUES`, clientTimeCol, deviceIdCol, deviceOsCol, sessionCol, sequenceCol,
		eventCol, paramIntCol, paramStrCol, ipCol, serverTimeCol)

	return err
}
