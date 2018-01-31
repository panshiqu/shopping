package spider

import (
	"log"
	"time"

	"github.com/panshiqu/framework/utils"
	"github.com/panshiqu/shopping/db"
)

// Spider 蜘蛛
type Spider struct {
}

var schedule *utils.Schedule

// StartSpider 开始
func StartSpider() {
	schedule = utils.NewSchedule(&Spider{})
	go schedule.Start()

	rows, err := db.Ins.Query("SELECT sku,priority FROM sku")
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		var sku, priority int64

		if err := rows.Scan(&sku, &priority); err != nil {
			log.Fatal(err)
		}

		schedule.Add(int(sku), time.Duration(priority)*time.Second, nil, true)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

// OnTimer 定时器到期
func (s *Spider) OnTimer(id int, parameter interface{}) {

}
