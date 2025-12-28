package ozon

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func setHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Origin", "https://www.ozon.ru")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

func warmUp(client *http.Client) error {
	req, err := http.NewRequest("GET", "https://www.ozon.ru/", nil)
	if err != nil {
		return err
	}
	setHeaders(req, "")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

func Ozon(query string) bool {
	searchpath := "/search?text=" + url.QueryEscape(query) + "&sorting=price&page=1"
	apiUrl := "https://api.ozon.ru/composer-api.bx/page/json/v2?url=" + url.QueryEscape(searchpath)
	referer := "https://www.ozon.ru/search/?text=" + url.QueryEscape(query)

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
		return false
	}

	current := apiUrl
	for step := 0; step < 25; step++ {
		fmt.Println("Step:", step)
		fmt.Println("URL:", current)
		seconds := rand.Intn(8) + 3
		time.Sleep(time.Duration(seconds) * time.Second)
		req, err := http.NewRequest("GET", current, nil)
		if err != nil {
			panic(err)
		}

		setHeaders(req, referer)
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		if resp.StatusCode == 301 || resp.StatusCode == 302 || resp.StatusCode == 303 || resp.StatusCode == 307 || resp.StatusCode == 308 {
			loc := resp.Header.Get("Location")
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if loc == "" {
				fmt.Println("redirect without location:", resp.StatusCode)
				return false
			}
			if strings.HasPrefix(loc, "/") {
				loc = "https://api.ozon.ru" + loc
			} else if strings.HasPrefix(loc, "composer-api") {
				loc = "https://api.ozon.ru/" + loc
			}
			current = loc
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			fmt.Println("read error:", readErr)
		}

		if resp.StatusCode == 200 {
			fmt.Println("OZON OK", resp.Status)
			os.WriteFile("ozon.json", body, 0644)
			return true
		}
		fmt.Println("Unexpected status:", resp.Status)
		if len(body) > 0 {
			s := string(body)
			if len(s) > 1000 {
				s = s[:1000]
			}
			fmt.Println("Body snippet:\n", s)
		}
	}
	return false
}

type Product struct {
	Link             string `json:"product_url:"`
	IMG              string `json:"image_url:"`
	ProductID        string `json:"product_id"`
	ProductName      string `json:"product_name"`
	DiscountPrice    string `json:"product_discount_price"` // цена со скидкой
	BasePrice        string `json:"product_base_price"`     // оригинальная цена
	ProductStatistic string `json:"product_statistic"`      // средняя оценка + количество отзывов
	ProductStars     string `json:"product_stars"`          // средняя оценка
	ProductReviews   string `json:"product_reviews"`        // количество отзывов
}

func getMainStateText(item gjson.Result, stateType string) gjson.Result {
	var found gjson.Result
	item.Get("mainState").ForEach(func(_, st gjson.Result) bool {
		if st.Get("type").String() == stateType {
			found = st
			return false
		}
		return true
	})
	return found
}

func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\u202F", " ")
	s = strings.ReplaceAll(s, "\u2009", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func Parse() {
	data, err := os.ReadFile("ozon.json")
	if err != nil {
		panic(err)
	}

	root := gjson.ParseBytes(data)

	var tileKey string
	root.Get("widgetStates").ForEach(func(k, _ gjson.Result) bool {
		if strings.HasPrefix(k.String(), "tileGridDesktop-") {
			tileKey = k.String()
			return false
		}
		return true
	})
	if tileKey == "" {
		fmt.Println("tileGridDesktop-* not found")
		return
	}

	tileStr := root.Get("widgetStates." + tileKey).String()
	tile := gjson.Parse(tileStr)

	items := tile.Get("items")
	if !items.Exists() || !items.IsArray() {
		fmt.Println("items not found or not array")
		return
	}

	fmt.Println("items:", len(items.Array()))

	products := make([]Product, 0, len(items.Array()))
	items.ForEach(func(_, item gjson.Result) bool {
		sku := item.Get("sku").String()
		link := item.Get("action.link").String()
		textAtom := getMainStateText(item, "textAtom")
		title := textAtom.Get("textAtom.text").String()
		if link != "" && strings.HasPrefix(link, "/") {
			link = "https://www.ozon.ru" + link
		}
		priceV2State := getMainStateText(item, "priceV2")
		priceV2 := priceV2State.Get("priceV2")

		var priceNow string
		var originalPrice string
		priceV2.Get("price").ForEach(func(_, price gjson.Result) bool {
			ts := price.Get("textStyle").String()
			txt := price.Get("text").String()

			switch ts {
			case "PRICE":
				priceNow = txt
			case "ORIGINAL_PRICE":
				originalPrice = txt
			}
			return true
		})
		priceNow = normalizeText(priceNow)
		originalPrice = normalizeText(originalPrice)

		img := item.Get("tileImage.items.0.image.link").String()

		var stars, reviews, statistic string
		item.Get("mainState").ForEach(func(_, st gjson.Result) bool {
			if st.Get("type").String() != "labelList" {
				return true
			}
			s := st.Get("labelList.items.0.title").String()
			r := st.Get("labelList.items.1.title").String()
			if s != "" && r != "" && strings.Contains(r, "отзыв") {
				stars = s
				reviews = r
				return false
			}
			return true
		})

		stars = normalizeText(stars)
		reviews = normalizeText(reviews)
		title = normalizeText(title)

		if stars != "" && reviews != "" {
			statistic = stars + " • " + reviews
		}

		p := Product{
			Link:             link,
			IMG:              img,
			ProductID:        sku,
			ProductName:      title,
			DiscountPrice:    priceNow,
			BasePrice:        originalPrice,
			ProductStatistic: statistic,
			ProductStars:     stars,
			ProductReviews:   reviews,
		}
		products = append(products, p)

		return true
	})

	out, err := json.MarshalIndent(products, "", "    ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("PRODUCTS_DATA.json", out, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("[+] Saved PRODUCTS_DATA.json:", len(products))
}
