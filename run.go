package kafcli

import "fmt"

// Run 执行完整的电子书转换流程。
func Run(book *Book, version, secret, measurement string) error {
	if err := book.Check(version); err != nil {
		return err
	}
	Analytics(version, secret, measurement, book.Format)
	book.ToString()
	if err := book.Parse(); err != nil {
		return fmt.Errorf("解析失败: %w", err)
	}
	book.Convert()
	return nil
}

// NewBookGUI 创建 GUI 模式下的默认 Book 配置。
func NewBookGUI(filename, cover string) *Book {
	book := &Book{
		Filename:   filename,
		DedupTitle: true,
		Tips:       true,
		Format:     "all",
		Lang:       "zh",
	}
	book.SetDefault()
	if cover != "" {
		book.Cover = cover
	} else {
		book.Cover = "none"
	}
	return book
}
