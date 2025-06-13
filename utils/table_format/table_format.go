package table_format

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
)

type Table struct {
	T table.Writer
}

// NewTable 格式化输出 @see https://blog.csdn.net/Meepoljd/article/details/129422612
//
// @return *Table
func NewTable() *Table {
	return &Table{
		T: table.NewWriter(),
	}
}

// Print 打印
//
//	@receiver d
func (d *Table) Print() {
	fmt.Println(d.T.Render())
}

// AddTitle 添加标题
//
//	@receiver d
//	@param title
func (d *Table) AddTitle(title string) {
	d.T.SetTitle(title)
}

// MakeHeader 设置头
//
//	@receiver d
func (d *Table) MakeHeader(header []interface{}) {

	d.T.AppendHeader(header)
	d.T.SetAutoIndex(true) // 自动标号
}

// AppendRows 追加行数据
//
//	@receiver d
//	@param rows
func (d *Table) AppendRows(rows []table.Row) {
	d.T.AppendRows(rows)
}

// AppendFooter 添加底部
//
//	@receiver d
//	@param footerText
func (d *Table) AppendFooter(footerText string) {
	d.T.AppendFooter(table.Row{footerText, footerText, footerText, footerText, footerText}, table.RowConfig{AutoMerge: true})
}
