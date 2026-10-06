# pet-battle-queue

宠物对战排队：请求按玩家 id 分片，各分片并发配对，最后汇总。只用标准库（`sync`、`sort` 等）。

```
go test ./...
go vet ./...
go run ./cmd/petqueue --sample=relax
go run ./cmd/petqueue --sample=dupe
go run ./cmd/petqueue --sample=order
go run ./cmd/petqueue --sample=cap
go run ./cmd/petqueue --sample=work
```

## 口径

- **分片**：玩家 id 的字符码求和后对分片数取模；配对只在同一分片内进行。
- **阈值**：评分差不超过 10 才配对；等待时长达到超时线的那一方，阈值放宽到 20。
- **名额**：每个分片每轮最多配 3 对，多出来的继续排队。
- **幂等**：同一个 `req_id` 只入队一次。
- **汇总**：所有分片的配对合并后，按对里第一个玩家 id 升序输出。

## 不变量

- 一个玩家在一轮里最多出现在一对里。
- 跨分片不会配对，分片内配对数不超过名额。
- `scanned` 不随队列长度平方级放大：两千条请求的 `scanned` 不超过 9000。

## 输出契约

配对结果打印一行 JSON：`pairs`（每项含 `a`、`b`）、`queued`、`scanned`。
`--sample=work` 只打印 `{"requests": N, "scanned": N}`。
