package wb

import (
	"agregator/adapters/models"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"github.com/tidwall/gjson"
)

func setHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.105 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Origin", "https://www.wildberries.ru")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

func warmUp(client *http.Client) error {
	req, err := http.NewRequest("GET", "https://www.wildberries.ru/", nil)
	if err != nil {
		return err
	}
	setHeaders(req, "")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return nil
}

func wildberries(query string) ([]byte, error) {
	apiUrl := "https://search.wb.ru/exactmatch/ru/common/v18/search?appType=1&curr=rub&dest=-1257786&lang=ru&page=1&query=" + url.QueryEscape(query) + "&resultset=catalog&sort=priceup&spp=30"
	referer := "https://www.wildberries.ru/catalog/0/search.aspx?search=" + url.QueryEscape(query)

	var body []byte

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка cookiejar:%w", err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	if err := warmUp(client); err != nil {
		return nil, fmt.Errorf("Warmup errors:%w", err)
	}

	rand.Seed(time.Now().UnixNano())
	for attempt := 0; attempt <= 20; attempt++ {
		req, err := http.NewRequest("GET", apiUrl, nil)
		if err != nil {
			fmt.Println("GET request err:", err)
			continue
		}
		setHeaders(req, referer)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Client error:", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("ошибка чтения resp.body")
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			sleepSecond := 2 * (1 << attempt)
			time.Sleep(time.Duration(sleepSecond) * time.Second)
			if len(body) > 0 {
				s := string(body)
				if len(s) > 500 {
					s = s[:500]
				}
				fmt.Println("resp status code != 200. last body snippet:", s)
			}
			continue
		}
		fmt.Println("WB OK\n", resp.Status)
		resp.Body.Close()
		break
	}
	return body, nil
}

func Parse(query string) ([]models.Product, error) {
	body, err := wildberries(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка сбора json WB:%w", err)
	}
	root := gjson.ParseBytes(body)
	products := root.Get("products")
	if !products.Exists() || !products.IsArray() {
		return nil, errors.New("items not found or not array")
	}

	var items []models.Product
	products.ForEach(func(_, products gjson.Result) bool {
		id := products.Get("id").Int()
		link := "https://www.wildberries.ru/catalog/" + strconv.FormatInt(id, 10) + "/detail.aspx"
		name := products.Get("name").String()
		discprice := (products.Get("sizes").Get("0").Get("price").Get("product").Int()) / 100
		baseprice := (products.Get("sizes").Get("0").Get("price").Get("basic").Int()) / 100
		stars := products.Get("rating").String()
		reviews := products.Get("feedbacks").String()
		var statistic string
		if stars != "" && reviews != "" {
			statistic = stars + " • " + reviews
		}
		var pic string
		for i := 0; i <= 30; i++ {
			vol := id / 100000
			part := id / 1000
			pic = "https://basket-" + strconv.Itoa(i) + ".wbbasket.ru" + "/vol" + strconv.FormatInt(vol, 10) + "/part" + strconv.FormatInt(part, 10) + "/" + strconv.FormatInt(id, 10) + "/images/big/1.webp"

			resp, err := http.Get(pic)
			if err != nil {
				continue
			}
			if resp.StatusCode != 200 {
				resp.Body.Close()
				continue
			} else {
				break
			}
		}
		p := models.Product{
			Link:             link,
			IMG:              pic,
			ProductID:        strconv.Itoa(int(id)),
			ProductName:      name,
			DiscountPrice:    strconv.Itoa(int(discprice)),
			BasePrice:        strconv.Itoa(int(baseprice)),
			ProductStatistic: statistic,
			ProductStars:     stars,
			ProductReviews:   reviews,
		}
		items = append(items, p)

		return true
	})
	return items, nil
}
