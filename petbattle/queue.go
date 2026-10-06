// Package petbattle 把宠物对战请求分片配对。
package petbattle

import (
	"sort"
	"sync"
)

// 排队参数。
const (
	Shards           = 4
	BaseThreshold    = 10
	RelaxedThreshold = 20
	TimeoutMS        = 1000
	MaxPairsPerShard = 3
)

// Request 是一条排队请求。
type Request struct {
	ID     string `json:"id"`
	Score  int    `json:"score"`
	WaitMS int    `json:"wait_ms"`
	ReqID  string `json:"req_id"`
}

// Pair 是一对配上的玩家。
type Pair struct {
	A string `json:"a"`
	B string `json:"b"`
}

// Counter 统计本轮比较过的请求条数。
type Counter struct {
	Scanned int
}

// Result 是一次配对的结果。
type Result struct {
	Pairs   []Pair `json:"pairs"`
	Queued  int    `json:"queued"`
	Scanned int    `json:"scanned"`
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ShardOf 按玩家 id 定分片。
func ShardOf(id string) int {
	sum := 0
	for _, r := range id {
		sum += int(r)
	}
	return sum % Shards
}

// Enqueue 把请求放进队列；同一个 req_id 只入队一次。
func Enqueue(queue []Request, req Request) []Request {
	if req.ReqID != "" {
		for _, q := range queue {
			if q.ReqID == req.ReqID {
				return queue
			}
		}
	}
	return append(queue, req)
}

// thresholdFor 返回这一对的评分差阈值：任一方等待达到超时线就放宽。
func thresholdFor(a, b Request) int {
	if a.WaitMS >= TimeoutMS || b.WaitMS >= TimeoutMS {
		return RelaxedThreshold
	}
	return BaseThreshold
}

func pairShard(reqs []Request, c *Counter) []Pair {
	var pairs []Pair
	used := make([]bool, len(reqs))
	for i := range reqs {
		if len(pairs) >= MaxPairsPerShard {
			break
		}
		if used[i] {
			continue
		}
		best := -1
		for j := range reqs {
			c.Scanned++
			if j == i || used[j] {
				continue
			}
			if abs(reqs[i].Score-reqs[j].Score) <= thresholdFor(reqs[i], reqs[j]) {
				if best == -1 || reqs[j].ID < reqs[best].ID {
					best = j
				}
			}
		}
		if best >= 0 {
			used[i], used[best] = true, true
			pairs = append(pairs, Pair{A: reqs[i].ID, B: reqs[best].ID})
		}
	}
	return pairs
}

// Match 分片并发配对，再汇总。
func Match(queue []Request, c *Counter) Result {
	byShard := make([][]Request, Shards)
	for _, req := range queue {
		s := ShardOf(req.ID)
		byShard[s] = append(byShard[s], req)
	}

	results := make([][]Pair, Shards)
	counters := make([]Counter, Shards)
	var wg sync.WaitGroup
	for s := 0; s < Shards; s++ {
		wg.Add(1)
		go func(s int) {
			defer wg.Done()
			results[s] = pairShard(byShard[s], &counters[s])
		}(s)
	}
	wg.Wait()

	var pairs []Pair
	for s := 0; s < Shards; s++ {
		pairs = append(pairs, results[s]...)
		c.Scanned += counters[s].Scanned
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].A < pairs[j].A })
	return Result{Pairs: pairs, Queued: len(queue), Scanned: c.Scanned}
}
