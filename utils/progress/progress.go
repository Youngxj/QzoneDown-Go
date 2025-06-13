package progress

import "fmt"

// 原生进度条
// see https://segmentfault.com/a/1190000023375330

type Bar struct {
	percent int64  // 百分比
	cur     int64  // 当前进度位置
	total   int64  // 总进度
	rate    string // 进度条
	graph   string // 显示符号
}

// NewOption 新配置
//
//	@receiver bar
//	@param start	开始数量
//	@param total	总数量
func (bar *Bar) NewOption(start, total int64) {
	bar.cur = start
	bar.total = total
	if bar.graph == "" {
		bar.graph = "█"
	}
	bar.percent = bar.getPercent()
	for i := 0; i < int(bar.percent); i += 2 {
		bar.rate += bar.graph // 初始化进度条位置
	}
}

// 获取百分比
//
//	@receiver bar
//	@return int64
func (bar *Bar) getPercent() int64 {
	if bar.total == 0 {
		return 0
	}
	return int64(float64(bar.cur) / float64(bar.total) * 100)
}

// Play 设置进度
//
//	@receiver bar
//	@param cur	当前进度数
func (bar *Bar) Play(cur int64) {
	bar.cur = cur
	bar.percent = bar.getPercent()
	bar.rate = ""
	for i := 0; i < int(bar.percent); i += 2 {
		bar.rate += bar.graph
	}
	fmt.Printf("\r[%-50s] %3d%%", bar.rate, bar.percent)
}

// Finish 完成进度
//
//	@receiver bar
func (bar *Bar) Finish() {
	fmt.Println()
}

// NewOptionWithGraph 新配置
//
//	@receiver bar
//	@param start	开始数量
//	@param total	总数量
//	@param graph	显示符号
func (bar *Bar) NewOptionWithGraph(start, total int64, graph string) {
	bar.graph = graph
	bar.NewOption(start, total)
}
