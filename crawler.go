package main

import (
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
	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}
	res, _ := client.Get("http://test.youplace.net/")
	urlSend := "http://test.youplace.net/question/1"
	res, _ = client.Get(urlSend)
	var body []byte
	for ; res.StatusCode == 200; {
		form := PageProcessor(res.Body)
		log.Println(form)
		res.Body.Close()
		res, _ = client.PostForm(res.Request.URL.String(), form)
	}
	log.Println(string(body))
	log.Println("StatusCode:", res.StatusCode)
	log.Println(res.Request.URL)
}

func NextToken(page *html.Tokenizer) (tag html.Token) {
	neededTokenTypes := map[atom.Atom]bool{
		atom.Form:   true,
		atom.Select: true,
		atom.Option: true,
		atom.Input:  true,
		atom.Title:  true,
		atom.A: true,
	}

	for {
		_ = page.Next()
		token := page.Token()
		if token.Type == html.ErrorToken {
			if page.Err() != io.EOF {
				log.Println("Ошибка парсинга тела http-ответа")
			}
			return token
		}

		if neededTokenTypes[token.DataAtom] && token.Type == html.StartTagToken {
			return token
		}
	}
}

func PageProcessor(body io.Reader) (form url.Values) {
	page := html.NewTokenizer(body)

	form = make(url.Values)
	var selectName string

	for {
		token := NextToken(page)
		if token.Type == html.ErrorToken {
			break
		}
		/*if token.Type == html.TextToken {
			text = fmt.Sprintf("%s%s", text, token.Data)
		}*/
		switch token.DataAtom {
		case atom.Input:
			name, inpType, value := InputProcessor(token)
			if inpType == "text" {
				form.Add(name, "test")
			} else if inpType == "radio" {
				radioValue, exists := form[name]
				if !exists || len(radioValue[0]) < len(value) {
					form.Set(name, value)
				}
			}
		case atom.Select:
			selectName, _, _ = InputProcessor(token)
		case atom.Option:
			_, _, value := InputProcessor(token)
			selectValue, exists := form[selectName]
			if !exists || len(selectValue[0]) < len(value) {
				form.Set(selectName, value)
			}
		case atom.Title:
			_ = page.Next()
			textToken := page.Token()
			if textToken.Type == html.TextToken {
				log.Println(textToken.Data)
			}
		case atom.A:
			
		}
	}
	return form
}

func InputProcessor(tag html.Token) (name, inpType, value string) {//пусть возвращает карту аттрибутов (ключ-значение)
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
