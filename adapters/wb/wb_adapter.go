package wb

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
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

func Wildberries(query string) {
	apiUrl := "https://search.wb.ru/exactmatch/ru/common/v18/search?appType=1&curr=rub&dest=-1257786&lang=ru&page=1&query=" + url.QueryEscape(query) + "&resultset=catalog&sort=priceup&spp=30"
	referer := "https://www.wildberries.ru/catalog/0/search.aspx?search=" + url.QueryEscape(query)

	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	if err := warmUp(client); err != nil {
		fmt.Println("Warmup errors:", err)
		return
	}

	rand.Seed(time.Now().UnixNano())
	for attempt := 0; attempt <= 30; attempt++ {
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

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			resp.Body.Close()
			sleepSecond := 2 * (1 << attempt)
			time.Sleep(time.Duration(sleepSecond) * time.Second)
			if len(body) > 0 {
				s := string(body)
				if len(s) > 500 {
					s = s[:500]
				}
				fmt.Println("last body snippet:", s)
			}
			continue
		}
		fmt.Println("WB OK", resp.Status)
		os.WriteFile("wb.json", body, 0644)
		resp.Body.Close()
		break
	}
}

func Parse() {
	data, err := os.ReadFile("wb.json")
	if err != nil {
		panic(err)
	}

	root := gjson.ParseBytes(data)
	products := root.Get("products")
	if !products.Exists() || !products.IsArray() {
		fmt.Println("items not found or not array")
		return
	}

	products.ForEach(func(_, products gjson.Result) bool {
		id := products.Get("id").Int()
		link := "https://www.wildberries.ru/catalog/" + strconv.FormatInt(id, 10) + "/detail.aspx"
		name := products.Get("name").String()
		price := products.Get("sizes").Get("0").Get("price").Get("product").Int()
		fmt.Println("Name:", name)
		fmt.Println("URL:", link)
		fmt.Println("Price: \n\n", price/100)
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
		fmt.Println("Image:", pic)
		return true
	})
}
