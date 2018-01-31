package spider

import (
	"github.com/panshiqu/framework/utils"
)

// Spider 蜘蛛
type Spider struct {
	schedule *utils.Schedule
}

// NewSpider 创建
func NewSpider() *Spider {
	s := &Spider{}
	s.schedule = utils.NewSchedule(s)
	go s.schedule.Start()
	return s
}

// OnTimer 定时器到期
func (s *Spider) OnTimer(id int, parameter interface{}) {

}
