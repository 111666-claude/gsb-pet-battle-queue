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

// canPair 判断两条请求能否配对：分差不超过 10；
// 任一方等待达到超时线时，阈值放宽到 20。
func canPair(a, b Request) bool {
	threshold := BaseThreshold
	if a.WaitMS >= TimeoutMS || b.WaitMS >= TimeoutMS {
		threshold = RelaxedThreshold
	}
	return abs(a.Score-b.Score) <= threshold
}

type entry struct {
	req  Request
	used bool
}

// pairShard 在单个分片内配对，最多配 MaxPairsPerShard 对。
// 按评分分桶，只比较阈值可达的桶，避免随队列长度平方级扫描。
func pairShard(reqs []Request, c *Counter) []Pair {
	byScore := make(map[int][]*entry)
	var scores []int
	for _, req := range reqs {
		if _, ok := byScore[req.Score]; !ok {
			scores = append(scores, req.Score)
		}
		e := &entry{req: req}
		byScore[req.Score] = append(byScore[req.Score], e)
	}
	sort.Ints(scores)
	for _, s := range scores {
		bucket := byScore[s]
		sort.Slice(bucket, func(i, j int) bool { return bucket[i].req.ID < bucket[j].req.ID })
	}

	var pairs []Pair
	for _, s := range scores {
		bucket := byScore[s]
		for _, e := range bucket {
			if len(pairs) >= MaxPairsPerShard {
				return pairs
			}
			if e.used {
				continue
			}
			var best *entry
			// 先找评分更低、id 最小的候选。
			for _, ls := range scores {
				if ls >= s {
					break
				}
				if s-ls > RelaxedThreshold {
					continue
				}
				for _, cand := range byScore[ls] {
					c.Scanned++
					if cand.used || !canPair(e.req, cand.req) {
						continue
					}
					if best == nil || cand.req.ID < best.req.ID {
						best = cand
					}
					break
				}
			}
			// 再在同分桶内找 id 更大的候选。
			for _, cand := range bucket {
				if cand.req.ID <= e.req.ID {
					continue
				}
				c.Scanned++
				if cand.used || !canPair(e.req, cand.req) {
					continue
				}
				if best == nil || cand.req.ID < best.req.ID {
					best = cand
				}
				break
			}
			if best != nil {
				e.used, best.used = true, true
				pairs = append(pairs, Pair{A: e.req.ID, B: best.req.ID})
			}
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
	return Result{Pairs: pairs, Queued: len(queue) - 2*len(pairs), Scanned: c.Scanned}
}
