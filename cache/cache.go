package cache

import (
	"database/sql"
	"sync"
	"time"

	"github.com/panshiqu/shopping/db"
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
func Update(id int64, price float64, content string, name string) (bool, error) {
	mtx.Lock()
	defer mtx.Unlock()

	args, ok := data[id]
	if !ok {
		args = &define.IndexArgs{
			SkuID: id,
			Name:  name,
		}

		if err := db.Ins.QueryRow("SELECT price,content FROM jd WHERE sku = ? ORDER BY id DESC LIMIT 1", args.SkuID).Scan(&args.Price, &args.Content); err != nil && err != sql.ErrNoRows {
			return false, err
		}

		if err := db.Ins.QueryRow("SELECT min_price,max_price,UNIX_TIMESTAMP(insert_timestamp) FROM sku WHERE sku = ?", args.SkuID).Scan(&args.MinPrice, &args.MaxPrice, &args.InsertTimestamp); err != nil {
			return false, err
		}

		if err := db.Ins.QueryRow("SELECT COUNT(*) FROM jd WHERE sku = ?", args.SkuID).Scan(&args.Sampling); err != nil {
			return false, err
		}

		data[args.SkuID] = args
	}

	args.Timestamp = time.Now().Format("01-02 15:04:05")

	if price == args.Price && content == args.Content {
		return false, define.ErrDataSame
	}

	var push bool

	if price == args.MinPrice && args.Price != args.MinPrice {
		push = true
	}

	if (price != -0.99) && (price < args.MinPrice || args.MinPrice == 0) {
		push = true
		args.MinPrice = price

		if _, err := db.Ins.Exec("UPDATE sku SET min_price = ? WHERE sku = ?", args.MinPrice, args.SkuID); err != nil {
			return false, err
		}
	}

	if price > args.MaxPrice || args.MaxPrice == 0 {
		args.MaxPrice = price

		if _, err := db.Ins.Exec("UPDATE sku SET max_price = ? WHERE sku = ?", args.MaxPrice, args.SkuID); err != nil {
			return false, err
		}
	}

	args.Price = price
	args.Content = content
	args.Sampling++
	return push, nil
}

// Select 查询
func Select(in []int64) (out []*define.IndexArgs) {
	mtx.RLock()
	defer mtx.RUnlock()
	now := time.Now().Unix()
	for _, v := range in {
		if va, ok := data[v]; ok {
			va.Duration = (time.Duration(now-va.InsertTimestamp) * time.Second).String()
			out = append(out, va)
		}
	}
	return
}

// Exist 存在
func Exist(in int64) bool {
	mtx.RLock()
	defer mtx.RUnlock()
	if _, ok := data[in]; ok {
		return true
	}
	if err := db.Ins.QueryRow("SELECT sku FROM sku WHERE sku = ?", in).Scan(&in); err != sql.ErrNoRows {
		return true
	}
	return false
}
