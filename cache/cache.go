package cache

import (
	"sync"

	"github.com/panshiqu/shopping/define"
)

var (
	mtx  sync.RWMutex
	data map[int64]*define.IndexArgs
)

func init() {
	data = make(map[int64]*define.IndexArgs)
}

// Update 更新
func Update(in *define.IndexArgs) {
	mtx.Lock()
	data[in.SkuID] = in
	mtx.Unlock()
}

// Select 查询
func Select(in []int64) (out []*define.IndexArgs) {
	mtx.RLock()
	defer mtx.RUnlock()
	for _, v := range in {
		if va, ok := data[v]; ok {
			out = append(out, va)
		}
	}
	return
}
