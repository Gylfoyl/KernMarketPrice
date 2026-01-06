package main

import (
	"agregator/adapters/models"
	"agregator/adapters/ozon"
	"agregator/adapters/wb"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Введите запрос:")
	if !scanner.Scan() {
		fmt.Fprint(os.Stderr, "Не удалось прочитать запрос")
		os.Exit(1)
	}
	query := scanner.Text()

	ok := ozon.Ozon(query)
	go wb.Wildberries(query)

	var itemsOzon []models.Product
	if ok == false {
		cmd := exec.Command("python", "./adapters/ozon/fallback.py", query)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("[!] Ошибка запуска Python скрипта:", err)
			os.Exit(1)
		}
		if len(output) == 0 {
			fmt.Fprintf(os.Stderr, "Python-скрипт вернул пустой вывод\n")
			os.Exit(1)
		}

		err = json.Unmarshal(output, &itemsOzon)
		if err != nil {
			fmt.Printf("[!] Ошибка парсинга JSON из Python:", err)
		}
		fmt.Printf("[+] Python фолбэк вернул %d товаров:", len(itemsOzon))
	} else {
		itemsOzon = ozon.Parse()
	}
	itemWB := wb.Parse()
	items := make([]models.Product, 0, len(itemsOzon)+len(itemWB))
	items = append(items, itemsOzon...)
	items = append(items, itemWB...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].DiscountPrice < items[j].DiscountPrice
	})
	out, err := json.MarshalIndent(items, "", "    ")
	if err != nil {
		fmt.Printf("[!] Ошибка сериализации JSON:", err)
		os.Exit(1)
	}
	err = os.WriteFile("PRODUCTS_DATA.json", out, 0644)
	if err != nil {
		fmt.Printf("[!] Ошибка записи в JSON файл:", err)
		os.Exit(1)
	}
	fmt.Printf("[+] Saved PRODUCTS_DATA.json (%d товаров)\n", len(items))
}
