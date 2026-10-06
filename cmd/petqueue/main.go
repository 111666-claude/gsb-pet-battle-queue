// Command petqueue 是宠物对战排队的场景入口。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/pet-battle-queue/petbattle"
)

func emit(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}

func main() {
	sample := flag.String("sample", "", "relax|dupe|order|cap|work")
	flag.Parse()

	switch *sample {
	case "relax":
		queue := []petbattle.Request{
			{ID: "p0", Score: 100, WaitMS: 2000, ReqID: "r1"},
			{ID: "p4", Score: 115, WaitMS: 0, ReqID: "r2"},
		}
		got := petbattle.Match(queue, &petbattle.Counter{})
		emit(map[string]int{"pairs": len(got.Pairs), "queued": got.Queued})
	case "dupe":
		queue := []petbattle.Request{}
		queue = petbattle.Enqueue(queue, petbattle.Request{ID: "p0", Score: 100, ReqID: "same"})
		queue = petbattle.Enqueue(queue, petbattle.Request{ID: "p0", Score: 100, ReqID: "same"})
		got := petbattle.Match(queue, &petbattle.Counter{})
		emit(map[string]int{"queued": got.Queued})
	case "order":
		queue := []petbattle.Request{
			{ID: "d0", Score: 100, ReqID: "r1"},
			{ID: "d4", Score: 100, ReqID: "r2"},
			{ID: "a0", Score: 100, ReqID: "r3"},
			{ID: "a4", Score: 100, ReqID: "r4"},
		}
		got := petbattle.Match(queue, &petbattle.Counter{})
		first := ""
		if len(got.Pairs) > 0 {
			first = got.Pairs[0].A
		}
		emit(map[string]any{"first": first, "pairs": len(got.Pairs)})
	case "cap":
		ids := []string{"p0", "p4", "p8", "t0", "t4", "t8", "x0", "x4"}
		queue := []petbattle.Request{}
		for i, id := range ids {
			queue = petbattle.Enqueue(queue, petbattle.Request{
				ID: id, Score: 100 + i, ReqID: fmt.Sprintf("r%d", i)})
		}
		got := petbattle.Match(queue, &petbattle.Counter{})
		emit(map[string]int{"pairs": len(got.Pairs)})
	case "work":
		names := []string{"p", "q", "r", "s"}
		queue := []petbattle.Request{}
		for i := 0; i < 2000; i++ {
			queue = petbattle.Enqueue(queue, petbattle.Request{
				ID:    fmt.Sprintf("%s%04d", names[i%4], i),
				Score: 100 + i%7,
				ReqID: fmt.Sprintf("w%d", i),
			})
		}
		got := petbattle.Match(queue, &petbattle.Counter{})
		emit(map[string]int{"requests": 2000, "scanned": got.Scanned})
	case "":
		flag.Usage()
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "未知场景："+*sample)
		os.Exit(2)
	}
}
