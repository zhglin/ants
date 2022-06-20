package ants

import "time"

// work池
type workerStack struct {
	items  []*goWorker
	expiry []*goWorker
	size   int
}

// 构建work池
func newWorkerStack(size int) *workerStack {
	return &workerStack{
		items: make([]*goWorker, 0, size),
		size:  size,
	}
}

// 现有worker数量
func (wq *workerStack) len() int {
	return len(wq.items)
}

// worker池是否为空
func (wq *workerStack) isEmpty() bool {
	return len(wq.items) == 0
}

// 添加worker
func (wq *workerStack) insert(worker *goWorker) error {
	wq.items = append(wq.items, worker)
	return nil
}

// 获取一个worker
func (wq *workerStack) detach() *goWorker {
	l := wq.len()
	if l == 0 {
		return nil
	}

	// 返回最后一个worker
	w := wq.items[l-1]
	wq.items[l-1] = nil // avoid memory leaks 避免内存泄漏
	wq.items = wq.items[:l-1]

	return w
}

func (wq *workerStack) retrieveExpiry(duration time.Duration) []*goWorker {
	n := wq.len()
	if n == 0 {
		return nil
	}

	expiryTime := time.Now().Add(-duration)
	index := wq.binarySearch(0, n-1, expiryTime)

	wq.expiry = wq.expiry[:0]
	if index != -1 {
		wq.expiry = append(wq.expiry, wq.items[:index+1]...)
		m := copy(wq.items, wq.items[index+1:])
		for i := m; i < n; i++ {
			wq.items[i] = nil
		}
		wq.items = wq.items[:m]
	}
	return wq.expiry
}

// 二分法查找
func (wq *workerStack) binarySearch(l, r int, expiryTime time.Time) int {
	var mid int
	for l <= r {
		mid = (l + r) / 2
		if expiryTime.Before(wq.items[mid].recycleTime) {
			r = mid - 1
		} else {
			l = mid + 1
		}
	}
	return r
}

// 重置
func (wq *workerStack) reset() {
	for i := 0; i < wq.len(); i++ {
		wq.items[i].task <- nil
		wq.items[i] = nil
	}
	wq.items = wq.items[:0]
}
