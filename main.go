package main

import (
	"agregator/adapters/models"
	"agregator/adapters/ozon"
	"agregator/adapters/wb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ctx         = context.Background()
	redisClient *redis.Client
)

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	for i := 0; i <= 3; i++ {
		_, err := redisClient.Ping(ctx).Result()
		if err == nil {
			log.Println("Подключено к Redis")
			return
		}
		log.Printf("Попытка %d подключиться к Redis не удалась:%v", i+1, err)
		time.Sleep(time.Second * 2)
	}
	log.Println("Работа без кэширования")
}

// нормализация запроса - не будет засорять бд. пример IPHONE    12 -> iphone 12
func normalizeQuery(q string) string {
	q = strings.TrimSpace(q)
	q = strings.ToLower(q)

	re := regexp.MustCompile(`\s+`)
	q = re.ReplaceAllString(q, " ")
	return q
}

func saveToRedis(query string, products []models.Product) error {
	data, err := json.Marshal(products)
	if err != nil {
		return err
	}
	query = normalizeQuery(query)
	err = redisClient.Set(ctx, query, data, 1*time.Hour).Err()
	return err
}

func getFromRedis(query string) ([]models.Product, error) {
	query = normalizeQuery(query)
	val, err := redisClient.Get(ctx, query).Result()
	if err == redis.Nil {
		return nil, errors.New("Ключа нет в redis")
	} else if err != nil {
		return nil, fmt.Errorf("Ошибка Redis:%w", err)
	}
	var products []models.Product
	err = json.Unmarshal([]byte(val), &products)
	return products, err
}

// парс
func searchProducts(query string) ([]models.Product, error) {
	query = normalizeQuery(query)
	cachedProducts, err := getFromRedis(query)
	if err == nil {
		fmt.Printf("Использован кэш для запроса %s", query)
		return cachedProducts, nil
	}

	var (
		wg        sync.WaitGroup
		itemsOzon []models.Product
		itemWB    []models.Product
		errOzon   error
		errWB     error
	)
	wg.Add(2)

	//ozon
	go func() {
		defer wg.Done()
		itemsOzon, errOzon = ozon.Parse(query)
		if errOzon != nil {
			cmd := exec.Command("python3", "./adapters/ozon/fallback.py", query)
			output, err := cmd.Output()
			if err != nil {
				errOzon = fmt.Errorf("Ошибка запуска скрипта:%w", err)
			}
			if len(output) == 0 {
				errOzon = fmt.Errorf("Скрипт вернул пустой вывод")
			}
			err = json.Unmarshal(output, &itemsOzon)
			if err != nil {
				errOzon = fmt.Errorf("Не удалось десериализовать ozon:%w", err)
			}
			fmt.Printf("Python фолбэк вернул %d товаров:", len(itemsOzon))
		}
	}()

	//wb
	go func() {
		defer wg.Done()
		itemWB, errWB = wb.Parse(query)
		if errWB != nil {
			errWB = fmt.Errorf("Не удалось спарсить wb:%w", errWB)
		}
	}()

	wg.Wait()

	if errOzon != nil && errWB != nil {
		return nil, fmt.Errorf("ошибка парсинга: ozon=%v wildberries=%v", errOzon, errWB)
	}

	items := append(itemsOzon, itemWB...)
	if len(items) == 0 {
		return nil, errors.New("товары не найдены")
	}

	sort.Slice(items, func(i, j int) bool {
		pi, _ := strconv.Atoi(items[i].DiscountPrice)
		pj, _ := strconv.Atoi(items[j].DiscountPrice)
		return pi < pj
	})

	err = saveToRedis(query, items)
	if err != nil {
		fmt.Printf("не удалось сохранить в Redis: %v", err)
	}

	return items, nil
}

func getProductsFromMemory(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	query = normalizeQuery(query)
	if query == "" {
		http.Error(w, "Параметр query обязателен", http.StatusBadRequest)
		return
	}

	products, err := getFromRedis(query)
	if err == redis.Nil {
		products = []models.Product{}
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(products)
	if err != nil {
		http.Error(w, "ошибка кодирования ответа", http.StatusInternalServerError)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	query = normalizeQuery(query)
	if query == "" {
		http.Error(w, "Параметр query обязателен", http.StatusBadRequest)
		return
	}

	products, err := searchProducts(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(products)
	if err != nil {
		http.Error(w, "ошибка кодирования ответа", http.StatusInternalServerError)
	}
}

func main() {
	initRedis()
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/products", getProductsFromMemory)
	fmt.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
