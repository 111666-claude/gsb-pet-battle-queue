package petbattle

import "testing"

func TestCloseScoresPairUp(t *testing.T) {
	queue := []Request{{ID: "p0", Score: 100}, {ID: "p4", Score: 105}}
	got := Match(queue, &Counter{})
	if len(got.Pairs) != 1 {
		t.Fatalf("分差在阈值内应配一对，得到 %+v", got)
	}
}

func TestFarScoresStayIdle(t *testing.T) {
	queue := []Request{{ID: "p0", Score: 0}, {ID: "p4", Score: 1000}}
	got := Match(queue, &Counter{})
	if len(got.Pairs) != 0 {
		t.Fatalf("分差过大不该配对，得到 %+v", got)
	}
}

func TestDifferentShardsStayApart(t *testing.T) {
	queue := []Request{{ID: "a0", Score: 100}, {ID: "b0", Score: 100}}
	if ShardOf("a0") == ShardOf("b0") {
		t.Fatal("样本玩家的分片应不同")
	}
	got := Match(queue, &Counter{})
	if len(got.Pairs) != 0 {
		t.Fatalf("跨分片不该配对，得到 %+v", got)
	}
}

func TestEmptyQueue(t *testing.T) {
	got := Match(nil, &Counter{})
	if len(got.Pairs) != 0 || got.Queued != 0 {
		t.Fatalf("空队列应无结果，得到 %+v", got)
	}
}

func TestEnqueueGrowsQueue(t *testing.T) {
	queue := Enqueue(nil, Request{ID: "p0"})
	if len(queue) != 1 {
		t.Fatalf("入队后应有一条，得到 %d", len(queue))
	}
}
