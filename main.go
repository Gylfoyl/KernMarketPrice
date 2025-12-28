package main

import (
	"agregator/adapters/ozon"
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
	if ok == false {
		cmd := exec.Command("python", "./adapters/ozon/fallback.py", query)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprint(os.Stderr, "Python fallback error:", err)
			os.Exit(1)
		}
	}

}
