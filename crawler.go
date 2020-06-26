package main

import (
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

var startUrl = "http://test.youplace.net" //адрес проходимого теста

type SearchArrayElement interface{}
type SearchArray []SearchArrayElement //массив с функцией поиска

/*
 * Проверяет, есть ли искомый элемент в массиве
 */
func (filter SearchArray) contains(elem SearchArrayElement) bool {
	for _, v := range filter {
		if v == elem {
			return true
		}
	}
	return false
}

func main() {
	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{//подключаем куки на клиенте
		Jar: cookieJar,
	}
	res := StartTest(client)
	if res == nil {
		return
	}
	neededTagTypes := SearchArray{atom.Form, atom.Title} //типы искомых тегов
	var err error
	for ; res.StatusCode == 200; {
		page := html.NewTokenizer(res.Body)
		tag := NextTag(page, neededTagTypes)
		if tag.DataAtom == atom.Title {//тег title может означать завершение теста
			_ = page.Next()
			textToken := page.Token()
			if textToken.Type == html.TextToken && textToken.Data == "Test successfully passed" {
				log.Println(textToken.Data) //успешное завершение теста
				break
			} else {
				log.Println("Page \"" + textToken.Data + "\" visited")
				tag = NextTag(page, SearchArray{atom.Form})
			}
		}
		if tag.DataAtom == atom.Form { //очередной вопрос с отправкой формы
			form := FormProcessor(page)
			err = res.Body.Close()
			if err != nil {
				log.Println(err)
				return
			}
			res, err = client.PostForm(res.Request.URL.String(), form)
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			log.Println("Unknown tag sequence")
			break
		}
	}
	err = res.Body.Close()
	if err != nil {
		log.Println(err)
		return
	}
}

/*
 * Инициализирует тест
 */
func StartTest(client *http.Client) (res *http.Response) {
	res, err := client.Get(startUrl) //получаем страницу начала теста
	if err != nil {
		log.Println(err)
		return
	}
	link := AttributesToMap( //извлечение ссылки на первый вопрос теста
		NextTag(
			html.NewTokenizer(res.Body),
			SearchArray{atom.A},
		),
	)["href"]
	if link == "" {
		log.Println("Href attribute is empty")
		return
	}
	res, err = client.Get(startUrl + link) //переход к первому вопросу
	if err != nil {
		log.Println(err)
		return
	}
	return res
}

/*
 * Получает следующий тег согласно переданному фильтру
 */
func NextTag(page *html.Tokenizer, neededTagTypes SearchArray) (tag html.Token) {
	for {
		_ = page.Next()
		tag = page.Token()
		if tag.Type == html.ErrorToken {
			if page.Err() != io.EOF {
				log.Println("Error on parsing response body")
			}
			return tag
		}

		if neededTagTypes.contains(tag.DataAtom) && tag.Type == html.StartTagToken {
			return tag
		}
	}
}

/*
 * Разбирает html-страницу на теги и возвращает форму ответа на тестовое задание
 */
func FormProcessor(page *html.Tokenizer) (form url.Values) {
	neededTagTypes := SearchArray{atom.Select, atom.Option, atom.Input}
	form = make(url.Values)
	var selectName string

	for {
		tag := NextTag(page, neededTagTypes)
		if tag.Type == html.ErrorToken {
			break
		}
		switch tag.DataAtom { //обрабатывает значения тегов формы
		case atom.Input:
			attributes := AttributesToMap(tag)
			switch attributes["type"] {
			case "text": //заменяет значение тега типа input-text на "test"
				form.Add(attributes["name"], "test")
			case "radio": //выбирает самое длинное значение для тегов типа input-radio с одним именем на форме
				value := attributes["value"]
				radioValue, exists := form[attributes["name"]]
				if !exists || len(radioValue[0]) < len(value) {
					form.Set(attributes["name"], value)
				}
			}
		case atom.Select: //получает имя тега типа select
			selectName = AttributesToMap(tag)["name"]
		case atom.Option: //выбирает самое длинное значение из тегов типа option из одного select
			value := AttributesToMap(tag)["value"]
			selectValue, exists := form[selectName]
			if !exists || len(selectValue[0]) < len(value) {
				form.Set(selectName, value)
			}
		}
	}
	return form
}

/*
 * Преобразует аттрибуты тега в карту ключ-значение для удобства
 */
func AttributesToMap(tag html.Token) map[string]string {
	filter := SearchArray{"type", "name", "value", "href"}
	attributes := make(map[string]string)
	for _, v := range tag.Attr {
		if filter.contains(v.Key) {
			attributes[v.Key] = v.Val
		}
	}
	return attributes
}
