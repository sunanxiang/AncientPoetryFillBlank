package spider

import (
	"net/http"
	"io/ioutil"
	"github.com/tidwall/gjson"
	"net/url"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

type BaiduSpider struct {
}

func (b *BaiduSpider) FindContent(name, author string) (searchResult SearchResult, err error) {
	baseUrl := "http://hanyu.baidu.com/hanyu/ajax/sugs?"

	client := &http.Client{
	}

	u, _ := url.Parse(baseUrl)
	q := u.Query()
	q.Set("mainkey", name)
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Add("User-Agent", `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36`)
	req.Header.Add("DNT", "1")
	req.Header.Add("Host", "hanyu.baidu.com")
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.8")
	req.Header.Add("Referer", "http://hanyu.baidu.com/shici/detail?pid=be520db056da43238035dc18bb1e1798&tn=sug_click")

	resp, errDo := client.Do(req)
	if errDo != nil || resp.StatusCode != 200 {
		err = errors.New("无法连接百度汉语 " + errDo.Error())
		return
	}

	robots, errBody := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if errBody != nil {
		err = errors.New("解析Body错误 " + errBody.Error())
		return
	}

	va := gjson.GetBytes(robots, "data.ret_array")
	if len(va.Array()) == 0 {
		//未找到结果
		err = errors.New("百度汉语搜索失败 -- 未找到结果 -- ")
		return
	}

	//如果有多个搜索结果，比对author是否正确
	va.ForEach(func(key, value gjson.Result) bool {
		//先看看有没有display_name
		displayName := value.Get("display_name.0").String()
		sid := value.Get("sid.0").String()
		if len(displayName) == 0 {
			//不是这条记录
			return true
		}

		//看看type
		typeStr := value.Get("type.0").String()
	    if typeStr == "poemline"{
			//取Source
			displayName = value.Get("source_poem.0").String()
			sid = value.Get("source_poem_sid.0").String()
		}

		literatureAuthor := value.Get("literature_author.0").String()
		//author是否一致
		if literatureAuthor == author {
			searchResult.Sid = sid
			searchResult.DisplayName = displayName
			searchResult.Author = literatureAuthor
			return false
		}
		return true // keep iterating
	})

	if len(searchResult.Sid) == 0{
		return searchResult,errors.New("未搜索到"+name)
	}
	return
}

func (b *BaiduSpider) GetContent(searchResult SearchResult) (content string, err error) {
	baseUrl := "http://hanyu.baidu.com/shici/detail"

	result := make([]string, 0, 0)
	client := &http.Client{
	}

	u, _ := url.Parse(baseUrl)
	q := u.Query()
	q.Set("pid", searchResult.Sid)
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Add("User-Agent", `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36`)
	req.Header.Add("DNT", "1")
	req.Header.Add("Host", "hanyu.baidu.com")
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.8")
	req.Header.Add("Referer", "http://hanyu.baidu.com/shici/detail?pid=be520db056da43238035dc18bb1e1798&tn=sug_click")

	resp, errDo := client.Do(req)

	if errDo != nil || resp.StatusCode != 200 {
		err = errors.New("无法连接百度汉语 " + errDo.Error())
		return
	}

	docm, errDoc := goquery.NewDocumentFromResponse(resp)

	if errDoc != nil {
		err = errors.New("解析Doc错误 " + errDoc.Error())
		return
	}

	pSelect := docm.Find("div.poem-detail-item-content .poem-detail-main-text")

	pSelect.Each(func(pos int, selection *goquery.Selection) {
		//跳过id为means_p的
		if att, ok := selection.Attr("id"); ok {
			if att == "means_p" {
				return
			}
		}
		content := strings.TrimSpace(selection.Text())
		result = append(result, content)
	})

	content = strings.Join(result, "")
	return
}
