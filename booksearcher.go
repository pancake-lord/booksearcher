package booksearcher

import (
  "fmt"
  "time"
  "image"
  "errors"
  "strconv"
  "strings"
  "net/http"
  "image/png"
  "image/jpeg"

  "golang.org/x/net/html"
)
// Book Object
// This contains a few private variables that are
// given accessor methods.
// This is the object that is created upon using
// the method "GetBooks"
type Book struct {
  isbn            [2]string
  title           string
  authors         []string
  publishers      []string
  edition         string
  binding         string
  cover           image.Image
  msrp            float32
  published       time.Time
}
func (b * Book) addISBN(isbn string) {
  if len(isbn) == 13 {
    b.isbn[1] = isbn
  } else if len(isbn) == 10 {
    b.isbn[0] = isbn
  }
}
// The 0 index has the ISBN 10 number
// The 1 index has the ISBN 13 number
func (b * Book) GetISBNs() [2]string {
  return b.isbn
}
func (b * Book) addTitle(t string) {
  b.title = t
}
// This gets the official title of the book
// being searched.
func (b * Book) GetTitle() string {
  return b.title
}
func (b * Book) addAuthors(auths []string) {
  for _, au := range auths {
    b.authors = append(b.authors, strings.TrimSpace(au))
  }
}
// This returns a slice of Authors. Some books
// have multiple authors.
func (b * Book) GetAuthors() []string {
  return b.authors
}
func (b * Book) addPublishers(publ []string) {
  for _, pbl := range publ {
    b.publishers = append(b.publishers, strings.TrimSpace(pbl))
  }
}
// This returns a slice of Publishers. Some books
// may have multiple publishers
func (b * Book) GetPublishers() []string {
  return b.publishers
}
func (b * Book) addEdition(edt string) {
  b.edition = strings.TrimSpace(edt)
}
// Returns the edition of the book. This is a string
// because not all editions are numerical.
func (b * Book) GetEdition() string {
  return b.edition
}
func (b * Book) addBinding(bind string) {
  b.binding = bind
}
func (b * Book) GetBinding() string {
  return b.binding
}
func (b * Book) addCover(imgUrl string) error {
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
// This returns an image.Image object. It is usually
// a smaller picture but roughly the size of a picture
// on a phone.
func (b * Book) GetCover() image.Image {
  return b.cover
}
func (b * Book) addPublishedDate(t string) {
  f, err := time.Parse("January 2006", strings.TrimSpace(t))
  if err != nil {
    panic(err)
  }
  b.published = f
}
// This returns a time.Time object. This will give you
// the month and year the book was published.
func (b * Book) GetPublishDate() time.Time {
  return b.published
}
func (b * Book) addPrice(p string) error {
  pr := strings.TrimSpace(strings.Replace(p, "$", "", 1))
  prc, err := strconv.ParseFloat(pr, 32)
  if err != nil {
    return err
  }
  b.msrp = float32(prc)
  return nil
}
// This gives you a float that represents the USD representation
// of the price of the book.
func (b * Book) GetPrice() float32 {
  return b.msrp
}
// This method creates and populates the book object. You
// cannot alter the book object in any way but you can
// access its information.
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
func getBookInfo(token * html.Tokenizer) (*Book, error) {
  var err error
  book := new(Book)
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
      case i == 5 && info:
        book.addBinding(tag.String())
      case i == 6 && info:
        book.addPublishers(strings.Split(tag.String(), ";"))
      case i == 7 && info:
        book.addPublishedDate(tag.String())
      case i == 8 && info:
        err = book.addPrice(tag.String())
      }
      info = false
    }
  }
  if err != nil {
    panic(err)
  }
  return book, err
}
