package main

import (
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func main() {
	cookieJar,_ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}
	res, _ := client.Get("http://test.youplace.net/")
	urlSend := "http://test.youplace.net/question/1"
	res, _ = client.Get(urlSend)
	var body []byte
	for ;res.StatusCode == 200; {
		form := PageProcessor(res.Body)
		log.Println(form)
		res.Body.Close()
		res, _ = client.PostForm(res.Request.URL.String(), form)
	}
	log.Println(string(body))
	log.Println("StatusCode:", res.StatusCode)
	log.Println(res.Request.URL)
}

func PageProcessor(body io.Reader) (form url.Values) {
	page := html.NewTokenizer(body)

	var start *html.Token
	var text string
	form = make(url.Values)
	var selectName string

	for {
		_ = page.Next()
		token := page.Token()
		if token.Type == html.ErrorToken {
			if page.Err() != io.EOF {
				fmt.Printf("Ошибка\n")
			}
			break
		}

		if start != nil && token.Type == html.TextToken {
			text = fmt.Sprintf("%s%s", text, token.Data)
		}

		if token.DataAtom == atom.Input {
			if token.Type == html.StartTagToken {
				name, inpType, value := InputProcessor(token)
				if inpType == "text" {
					form.Add(name, "test")
				} else if inpType == "radio" {
					radioValue, exists := form[name]
					if !exists || len(radioValue[0]) < len(value) {
						form.Set(name, value)
					}
				}
			}
		} else if token.DataAtom == atom.Select && token.Type == html.StartTagToken {
			selectName, _, _ = InputProcessor(token)
		}
		if token.DataAtom == atom.Option {
			switch token.Type {
			case html.StartTagToken:
				if len(token.Attr) > 0 {
					start = &token
				}
			case html.EndTagToken:
				if start == nil {
					continue
				}
				_, _, value := InputProcessor(*start)
				selectValue, exists := form[selectName]
				if !exists || len(selectValue[0]) < len(value) {
					form.Set(selectName, value)
				}

				start = nil
				text = ""
			}
		}
	}
	return form
}

func InputProcessor(tag html.Token) (name, inpType, value string) {
	for i := range tag.Attr {
		if tag.Attr[i].Key == "type" {
			inpType = strings.TrimSpace(tag.Attr[i].Val)
		} else if tag.Attr[i].Key == "name" {
			name = strings.TrimSpace(tag.Attr[i].Val)
		} else if tag.Attr[i].Key == "value" {
			value = strings.TrimSpace(tag.Attr[i].Val)
		}
	}
	return name, inpType, value
}
