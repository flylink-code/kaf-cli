package kafcli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmaupin/go-epub"
)

type EpubConverter struct{}

func (convert EpubConverter) wrapTitle(title, content string) string {
	var buff bytes.Buffer
	buff.WriteString(htmlTitleStart)
	buff.WriteString(title)
	buff.WriteString(htmlTitleEnd)
	buff.WriteString(content)
	return buff.String()
}

func (convert EpubConverter) Build(book Book) error {
	fmt.Println("正在生成epub")
	start := time.Now()
	// 写入样式
	tempDir, err := os.MkdirTemp("", "kaf-cli")
	if err != nil {
		return fmt.Errorf("创建临时文件夹失败: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create a ne EPUB
	e := epub.NewEpub(book.Bookname)
	e.SetLang(book.Lang)
	// Set the author
	e.SetAuthor(book.Author)
	// AI 生成的简介（如有）写入 epub 描述字段
	if book.AISummary != "" {
		e.SetDescription(book.AISummary)
	}

	pageStylesFile := filepath.Join(tempDir, "page_styles.css")
	var epubcss = cssContent
	var excss string
	if book.LineHeight != "" {
		excss = fmt.Sprintf("line-height: %s;", book.LineHeight)
	}
	if b, _ := isExists(book.Font); b {
		fontfile, err := e.AddFont(book.Font, "")
		if err != nil {
			return fmt.Errorf("添加字体失败: %w", err)
		}
		excss += `
font-family: "embedfont";
`
		epubcss += fmt.Sprintf(`
@font-face {
  font-family: "embedfont";
  src: url(%s) format('truetype');
}
`, fontfile)
	}

	err = os.WriteFile(pageStylesFile, []byte(fmt.Sprintf(epubcss, book.Align, book.Bottom, book.Indent, excss)), 0666)
	if err != nil {
		return fmt.Errorf("无法写入样式文件: %w", err)
	}
	css, err := e.AddCSS(pageStylesFile, "")
	if err != nil {
		return fmt.Errorf("无法写入样式文件: %w", err)
	}

	if book.Cover != "" {
		coverExt := strings.ToLower(filepath.Ext(book.Cover))
		if coverExt == "" {
			coverExt = ".png"
		}
		img, err := e.AddImage(book.Cover, "cover"+coverExt)
		if err != nil {
			return fmt.Errorf("添加封面失败: %w", err)
		}
		e.SetCover(img, "")
	}

	for _, section := range book.SectionList {
		if len(section.Sections) > 0 {
			internalFilename, err := e.AddSection(
				convert.wrapTitle(section.Title, section.Content),
				section.Title,
				"",
				css,
			)
			if err != nil {
				return fmt.Errorf("添加章节失败: %w", err)
			}
			for _, subsecton := range section.Sections {
				if _, err := e.AddSubSection(
					internalFilename,
					convert.wrapTitle(subsecton.Title, subsecton.Content),
					subsecton.Title,
					"",
					css,
				); err != nil {
					return fmt.Errorf("添加子章节失败: %w", err)
				}
			}
		} else {
			if _, err := e.AddSection(convert.wrapTitle(section.Title, section.Content), section.Title, "", css); err != nil {
				return fmt.Errorf("添加章节失败: %w", err)
			}
		}
	}

	// Write the EPUB
	fmt.Println("正在生成电子书...")
	epubName := book.Out + ".epub"
	err = e.Write(epubName)
	if err != nil {
		return fmt.Errorf("写入epub失败: %w", err)
	}
	// 计算耗时
	end := time.Now().Sub(start)
	fmt.Println("生成EPUB电子书耗时:", end)
	return nil
}
