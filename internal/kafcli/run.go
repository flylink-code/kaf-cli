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
	if err := book.Convert(); err != nil {
		return fmt.Errorf("转换失败: %w", err)
	}
	return nil
}

// GUIOptions 图形界面转换选项。
type GUIOptions struct {
	Filename        string
	Cover           string
	Author          string
	Format          string
	Match           string
	VolumeMatch     string
	DedupTitle      bool
	Tips            bool
	NormalizeQuotes bool
}

// NewBookGUI 根据 GUI 选项创建 Book。
func NewBookGUI(opts GUIOptions) *Book {
	book := &Book{
		Filename:        opts.Filename,
		DedupTitle:      opts.DedupTitle,
		Tips:            opts.Tips,
		NormalizeQuotes: opts.NormalizeQuotes,
		Format:          opts.Format,
		Lang:            "zh",
	}
	if opts.Author != "" {
		book.Author = opts.Author
	}
	if opts.Match != "" {
		book.Match = opts.Match
	}
	if opts.VolumeMatch != "" {
		book.VolumeMatch = opts.VolumeMatch
	}
	book.SetDefault()
	if book.Format == "" {
		book.Format = "all"
	}
	if opts.Cover != "" {
		book.Cover = opts.Cover
	} else {
		book.Cover = "none"
	}
	return book
}
