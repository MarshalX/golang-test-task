## Golang test task

Vacancy: backend developer of high load infrastructure.

### Task

Write a high-performance server that will accept a batch of events, enrich it with data, and write to ClickHouse.
Average number of events per batch: 30. Minimal RPS: 200. Events are separated by `\n` symbol in the batch. Data to
enrich: client IP, server time.

Example of **formatted** payload:

```json
{
  "client_time": "2022-12-01 23:59:00",
  "device_id": "8273D9AA-4ADF-4B37-A60F-3E9E645C821E",
  "device_os": "iOS 16.1.0",
  "session": "bmwRi8mAUypxjpaH",
  "sequence": 1,
  "event": "app_start",
  "param_int": 0,
  "param_str": "some text"
}
```

### Solution

Endpoint: POST `/submit`.

> We recommend inserting data in packets of at least 1000 rows, or no more than a single request per second.

Source: https://clickhouse.com/docs/en/about-us/performance/#performance-when-inserting-data

This recommendation is one of the reasons why `inMemoryStorage` was created.

The library _chconn_ (ClickHouse low-level Driver) was picked for high-performance bulk inserts.
[Benchmarks](https://github.com/vahid-sohrabloo/chconn#benchmarks)

`zap.Logger` is used for logs instead of `zap.SugaredLogger` to get even faster logging and fewer allocations.

Run ClickHouse with default configuration without password and expose ports:

```shell
docker run -d -p8123:8123 -p9000:9000 --name analytics-clickhouse-server --ulimit nofile=262144:262144 clickhouse/clickhouse-server
```

Curl for local tests:

```shell
curl --request POST \
  --url http://localhost:3000/submit \
  --header 'Content-Type: application/json' \
  --data '{"client_time": "2022-12-01 23:59:00", "device_id": "8273D9AA-4ADF-4B37-A60F-3E9E645C821E", "device_os": "iOS 16.1.0","session": "bmwRi8mAUypxjpaH", "sequence": 1, "event": "app_start", "param_int": 0, "param_str": "some text"}'
```
