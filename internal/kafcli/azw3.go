package kafcli

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"os"
	"time"

	"github.com/leotaku/mobi"
	_ "golang.org/x/image/webp"
	"golang.org/x/text/language"
)

type Azw3Converter struct{}

func (convert Azw3Converter) Build(book Book) error {
	fmt.Println("使用第三方库生成azw3, 不保证所有样式都能正常显示")
	fmt.Println("正在生成azw3...")
	start := time.Now()
	chunks := SectionSliceChunk(book.SectionList, 2000)
	for i, chunk := range chunks {
		index := i + 1
		title := fmt.Sprintf("%s_%d", book.Bookname, index)
		filename := fmt.Sprintf("%s_%d.azw3", book.Out, index)
		if len(chunks) == 1 {
			title = fmt.Sprintf("%s", book.Bookname)
			filename = fmt.Sprintf("%s.azw3", book.Out)
		}
		mb := mobi.Book{
			Title:       title,
			Authors:     []string{book.Author},
			CreatedDate: time.Now(),
			Chapters:    []mobi.Chapter{},
			Language:    language.MustParse(book.Lang),
			UniqueID:    rand.Uint32(),
		}
		var excss string
		if book.LineHeight != "" {
			excss = fmt.Sprintf("line-height: %s;", book.LineHeight)
		}
		css := fmt.Sprintf(cssContent, book.Align, book.Bottom, book.Indent, excss)
		for _, section := range chunk {
			ch := mobi.Chapter{
				Title:  section.Title,
				Chunks: mobi.Chunks(convert.wrapTitle(section.Title, section.Content, book.Align)),
			}
			mb.Chapters = append(mb.Chapters, ch)
			if len(section.Sections) > 0 {
				for _, subsection := range section.Sections {
					ch := mobi.Chapter{
						Title:  subsection.Title,
						Chunks: mobi.Chunks(convert.wrapTitle(subsection.Title, subsection.Content, book.Align)),
					}
					mb.Chapters = append(mb.Chapters, ch)
				}
			}
		}

		mb.CSSFlows = []string{css}
		if book.Cover != "" {
			img, err := loadCoverImage(book.Cover)
			if err != nil {
				fmt.Printf("[警告] AZW3封面加载失败，将跳过封面: %v\n", err)
			} else {
				mb.CoverImage = img
			}
		}

		// Convert book to PalmDB database
		db := mb.Realize()

		// Write database to file
		f, _ := os.Create(filename)
		err := db.Write(f)
		if err != nil {
			return fmt.Errorf("保存失败: %w", err)
		}
	}

	fmt.Println("生成azw3电子书耗时:", time.Now().Sub(start))
	return nil
}

func (convert Azw3Converter) wrapTitle(title, content, align string) string {
	var buff bytes.Buffer
	buff.WriteString(fmt.Sprintf(mobiTtmlTitleStart, align))
	buff.WriteString(title)
	buff.WriteString(htmlTitleEnd)
	buff.WriteString(content)
	return buff.String()
}

func SectionSliceChunk(s []Section, size int) [][]Section {
	var ret [][]Section
	for size < len(s) {
		// s[:size:size] 表示 len 为 size，cap 也为 size，第二个冒号后的 size 表示 cap
		s, ret = s[size:], append(ret, s[:size:size])
	}
	ret = append(ret, s)
	return ret
}

func loadCoverImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}
