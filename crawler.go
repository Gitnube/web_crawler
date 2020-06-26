package main

import (
	"bytes"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

var startUrl = "http://test.youplace.net"

func main() {
	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}
	res := StartTest(client)
	if res == nil {
		return
	}
	neededTokenTypes := map[atom.Atom]bool{
		atom.Form:  true,
		atom.Title: true,
	}
	for ; res.StatusCode == 200; {
		body, _ := ioutil.ReadAll(res.Body)
		err := res.Body.Close()
		if err != nil {
			log.Println(err)
			return
		}
		page := html.NewTokenizer(bytes.NewReader(body))
		tag := NextTag(page, neededTokenTypes)
		if tag.DataAtom == atom.Title {
			_ = page.Next()
			textToken := page.Token()
			if textToken.Type == html.TextToken && textToken.Data == "Test successfully passed" {
				log.Println(textToken.Data)
				break
			} else {
				log.Println("Page \"" + textToken.Data + "\" visited")
				tag = NextTag(page, map[atom.Atom]bool{atom.Form: true})
			}
		}
		switch tag.DataAtom {
		case atom.Form:
			form := FormProcessor(page)
			res, err = client.PostForm(res.Request.URL.String(), form)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func StartTest(client *http.Client) (res *http.Response) {
	res, err := client.Get(startUrl)
	if err != nil {
		log.Println(err)
		return
	}
	link := AttributesToMap(
		NextTag(
			html.NewTokenizer(res.Body),
			map[atom.Atom]bool{atom.A: true},
		),
	)["href"]
	if link == "" {
		log.Println("Href attribute is empty")
		return
	}
	res, err = client.Get(startUrl + link)
	if err != nil {
		log.Println(err)
		return
	}
	return res
}

func NextTag(page *html.Tokenizer, neededTokenTypes map[atom.Atom]bool) (tag html.Token) {

	for {
		_ = page.Next()
		token := page.Token()
		if token.Type == html.ErrorToken {
			if page.Err() != io.EOF {
				log.Println("Error on parsing response body")
			}
			return token
		}

		if neededTokenTypes[token.DataAtom] && token.Type == html.StartTagToken {
			return token
		}
	}
}

func FormProcessor(page *html.Tokenizer) (form url.Values) {

	neededTokenTypes := map[atom.Atom]bool{
		atom.Select: true,
		atom.Option: true,
		atom.Input:  true,
	}
	form = make(url.Values)
	var selectName string

	for {
		token := NextTag(page, neededTokenTypes)
		if token.Type == html.ErrorToken {
			break
		}
		switch token.DataAtom {
		case atom.Input:
			attributes := AttributesToMap(token)
			switch attributes["type"] {
			case "text":
				form.Add(attributes["name"], "test")
			case "radio":
				value := attributes["value"]
				radioValue, exists := form[attributes["name"]]
				if !exists || len(radioValue[0]) < len(value) {
					form.Set(attributes["name"], value)
				}
			}
		case atom.Select:
			selectName = AttributesToMap(token)["name"]
		case atom.Option:
			value := AttributesToMap(token)["value"]
			selectValue, exists := form[selectName]
			if !exists || len(selectValue[0]) < len(value) {
				form.Set(selectName, value)
			}
		}
	}
	return form
}

func AttributesToMap(tag html.Token) map[string]string {
	filter := map[string]bool{"type": true, "name": true, "value": true, "href": true}
	attributes := make(map[string]string)
	for _, v := range tag.Attr {
		if filter[v.Key] {
			attributes[v.Key] = v.Val
		}
	}
	return attributes
}
