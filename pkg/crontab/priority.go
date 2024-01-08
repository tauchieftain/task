package crontab

import (
	"container/heap"
)

type PriorityItem struct {
	Value    interface{}
	Priority int
	Index    int
}

type PriorityQueue []*PriorityItem

func (pq *PriorityQueue) Len() int {
	return len(*pq)
}

func (pq *PriorityQueue) Less(i, j int) bool {
	return (*pq)[i].Priority < (*pq)[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].Index = i
	(*pq)[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*PriorityItem)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	item := (*pq)[len(*pq)-1]
	item.Index = -1
	*pq = (*pq)[0 : len(*pq)-1]
	return item
}

func (pq *PriorityQueue) Update(item *PriorityItem, value interface{}, priority int) {
	item.Value = value
	item.Priority = priority
	heap.Fix(pq, item.Index)
}
