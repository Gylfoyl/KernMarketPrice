import pandas as pd
import requests

url = 'https://www.wildberries.ru/__internal/u-search/exactmatch/ru/common/v18/search?ab_testing=false&ab_testing=false&appType=1&curr=rub&dest=12358312&hide_dtype=9&hide_vflags=4294967296&inheritFilters=false&lang=ru&page=1&query=%D0%BD%D0%B0%D1%83%D1%88%D0%BD%D0%B8%D0%BA%D0%B8%20hiper%20x&resultset=catalog&sort=popular&spp=30&suppressSpellcheck=false'

try:
    # Отправляем GET-запрос
    response = requests.get(url)
    # Проверяем, успешен ли запрос (код 200)
    response.raise_for_status()
    # Выводим текст (содержимое) страницы
    print(response.text)
except requests.exceptions.RequestException as e:
    print(f"Произошла ошибка при запросе: {e}")