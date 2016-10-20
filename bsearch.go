package bsearch

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Book struct {
	isbn       [2]string
	title      string
	authors    []string
	publishers []string
	edition    string
	cover      image.Image
	msrp       float32
	Comm       Comm
	Expiration time.Time
}

func (b *Book) addISBN(isbn string) {
	if len(isbn) == 13 {
		b.isbn[1] = isbn
	} else if len(isbn) == 10 {
		b.isbn[0] = isbn
	}
}
func (b *Book) addTitle(t string) {
	b.title = t
}
func (b *Book) GetTitle() string {
	return b.title
}
func (b *Book) addAuthors(auths []string) {
	for _, au := range auths {
		b.authors = append(b.authors, strings.TrimSpace(au))
	}
}
func (b *Book) GetAuthors() []string {
	return b.authors
}
func (b *Book) addPublishers(publ []string) {
	for _, pbl := range publ {
		b.publishers = append(b.publishers, strings.TrimSpace(pbl))
	}
}
func (b *Book) GetPublishers() []string {
	return b.publishers
}
func (b *Book) addEdition(edt string) {
	b.edition = strings.TrimSpace(edt)
}
func (b *Book) GetEdition() string {
	return b.edition
}
func (b *Book) addCover(imgUrl string) error {
	response, err := http.Get(imgUrl)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New(response.Status)
	}
	var img image.Image
	if strings.HasSuffix(imgUrl, ".jpg") {
		img, err = jpeg.Decode(response.Body)
	} else if strings.HasSuffix(imgUrl, ".png") {
		img, err = png.Decode(response.Body)
	}
	if err != nil {
		return err
	}
	b.cover = img
	return nil
}
func (b *Book) GetCover() image.Image {
	return b.cover
}
func (b *Book) addPrice(p string) error {
	pr := strings.TrimSpace(strings.Replace(p, "$", "", 1))
	prc, err := strconv.ParseFloat(pr, 32)
	if err != nil {
		return err
	}
	b.msrp = float32(prc)
	return nil
}
func (b *Book) GetPrice() float32 {
	return b.msrp
}

type Comm struct {
	Amazon struct {
		New  float32 `json:"new"`
		Used float32 `json:"used"`
		Rent float32 `json:"rent"`
	}
	Barnes struct {
		New  float32 `json:"new"`
		Used float32 `json:"used"`
		Rent float32 `json:"rent"`
	}
	BookRenter struct {
		New  float32 `json:"new"`
		Used float32 `json:"used"`
		Rent float32 `json:"rent"`
	}
	Chegg struct {
		New  float32 `json:"new"`
		Used float32 `json:"used"`
		Rent float32 `json:"rent"`
	}
}

func GetBook(isbn string) (*Book, error) {
	url := fmt.Sprintf("http://www.isbnsearch.org/isbn/%s", isbn)
	response, err := http.Get(url)
	if err != nil {
		return &Book{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &Book{}, errors.New(response.Status)
	}
	token := html.NewTokenizer(response.Body)
	return getBookInfo(token)
}
func getBookInfo(token *html.Tokenizer) (*Book, error) {
	var err error
	book := new(Book)
	book.Expiration = time.Now().Add(5 * 8766 * time.Hour)
	var clear, binfo, info bool
	i := 0
	for !clear {
		t := token.Next()
		if t == html.StartTagToken {
			tag := token.Token()
			if tag.Data == "div" {
				for _, i := range tag.Attr {
					if i.Key == "class" && i.Val == "bookinfo" {
						binfo = true
						fmt.Println("Made it to the info")
					}
				}
			}
			if tag.Data == "h2" && binfo {
				info = true
				continue
			}
			if tag.Data == "p" && binfo && len(tag.Attr) > 0 {
				clear = true
			}
			if tag.Data == "p" && binfo && len(tag.Attr) == 0 {
				info = false
				i++
				continue
			}
			if tag.Data == "a" && binfo && i < 3 {
				info = true
				continue
			}
		}
		if t == html.SelfClosingTagToken {
			tag := token.Token()
			if tag.Data == "img" {
				for _, cov := range tag.Attr {
					if cov.Key == "src" {
						err = book.addCover(cov.Val)
						break
					}
				}
			}
		}
		if t == html.EndTagToken && binfo {
			tag := token.Token()
			if tag.Data == "strong" && i > 2 {
				info = true
			}
		}
		if t == html.TextToken && binfo {
			tag := token.Token()
			switch {
			case i == 0 && info:
				book.addTitle(tag.String())
			case i == 1 && info:
				book.addISBN(tag.String())
			case i == 2 && info:
				book.addISBN(tag.String())
			case i == 3 && info:
				au := strings.Split(tag.String(), ";")
				book.addAuthors(au)
			case i == 4 && info:
				book.addEdition(tag.String())
			case i == 6 && info:
				book.addPublishers(strings.Split(tag.String(), ";"))
			case i == 8 && info:
				err = book.addPrice(tag.String())
			}
			info = false
		}
	}
	if err != nil {
		panic(err)
	}
	e := getPrices(token, book)
	if e != nil && err == nil {
		err = e
	}
	if err != nil {
		panic(err)
	}
	return book, err
}
func getPrices(token *html.Tokenizer, b *Book) error {
	var err error
	var end bool
	var amazon, brenter, barnes, chegg bool
	index := 0
	for !end {
		t := token.Next()
		if t == html.StartTagToken {
			tag := token.Token()
			if tag.Data == "div" {
				for _, i := range tag.Attr {
					if i.Key == "id" && i.Val == "footer" {
						end = true
						break
					}
				}
				continue
			}
			if tag.Data == "th" {
				index++
				continue
			}
		}
		if t == html.SelfClosingTagToken {
			tag := token.Token()
			if tag.Data == "img" {
				for _, alt := range tag.Attr {
					if alt.Key == "alt" {
						switch {
						case strings.Contains(alt.Val, "Amazon"):
							amazon = true
						case strings.Contains(alt.Val, "Book Renter"):
							brenter = true
						case strings.Contains(alt.Val, "Chegg"):
							chegg = true
						case strings.Contains(alt.Val, "Barnes"):
							barnes = true
						}
					}
				}
				continue
			}
		}
		if t == html.TextToken {
			tag := token.Token()

			switch {
			case amazon:
				price := strings.TrimPrefix(tag.String(), "$")
				price = strings.TrimSpace(price)
				if price == "" {
					continue
				}
				pri, err := strconv.ParseFloat(price, 32)
				if err != nil {
					panic(err)
				}
				pr := float32(pri)

				switch index {
				case 1:
					b.Comm.Amazon.New = pr
				case 2:
					b.Comm.Amazon.Used = pr
				case 3:
					b.Comm.Amazon.Rent = pr
				}
				amazon = false
			case barnes:
				price := strings.TrimPrefix(tag.String(), "$")
				price = strings.TrimSpace(price)
				if price == "" {
					continue
				}
				pri, err := strconv.ParseFloat(price, 32)
				if err != nil {
					panic(err)
				}
				pr := float32(pri)

				switch index {
				case 1:
					b.Comm.Barnes.New = pr
				case 2:
					b.Comm.Barnes.Used = pr
				case 3:
					b.Comm.Barnes.Rent = pr
				}
				barnes = false
			case brenter:
				price := strings.TrimPrefix(tag.String(), "$")
				price = strings.TrimSpace(price)
				if price == "" {
					continue
				}
				pri, err := strconv.ParseFloat(price, 32)
				if err != nil {
					panic(err)
				}
				pr := float32(pri)

				switch index {
				case 1:
					b.Comm.BookRenter.New = pr
				case 2:
					b.Comm.BookRenter.Used = pr
				case 3:
					b.Comm.BookRenter.Rent = pr
				}
				brenter = false
			case chegg:
				price := strings.TrimPrefix(tag.String(), "$")
				price = strings.TrimSpace(price)
				if price == "" {
					continue
				}
				pri, err := strconv.ParseFloat(price, 32)
				if err != nil {
					panic(err)
				}
				pr := float32(pri)

				switch index {
				case 1:
					b.Comm.Chegg.New = pr
				case 2:
					b.Comm.Chegg.Used = pr
				case 3:
					b.Comm.Chegg.Rent = pr
				}
				chegg = false
			}
		}
	}
	return err
}
