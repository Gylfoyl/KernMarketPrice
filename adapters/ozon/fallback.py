import time as tm
from bs4 import BeautifulSoup
from selenium.webdriver.common.by import By
import json
import sys
import time
import undetected_chromedriver as uc 
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC 
from selenium.webdriver.support.ui import WebDriverWait
from urllib.parse import quote

def page_down(driver):
    driver.execute_script('''
                            const scrollStep = 200; // Размер шага прокрутки (в пикселях)
                            const scrollInterval = 100; // Интервал между шагами (в миллисекундах)

                            const scrollHeight = document.documentElement.scrollHeight;
                            let currentPosition = 0;
                            const interval = setInterval(() => {
                                window.scrollBy(0, scrollStep);
                                currentPosition += scrollStep;

                                if (currentPosition >= scrollHeight) {
                                    clearInterval(interval);
                                }
                            }, scrollInterval);
                        ''')


def collect_product_info(driver, url=''):

    driver.switch_to.new_window('tab')

    tm.sleep(3)
    driver.get(url=url)
    tm.sleep(3)

    # product_id
    product_id = driver.find_element(
        By.XPATH, '//div[contains(text(), "Артикул: ")]'
    ).text.split('Артикул: ')[1]

    # print(product_id)

    page_source = str(driver.page_source)
    soup = BeautifulSoup(page_source, 'lxml')

    with open(f'product_{product_id}.html', 'w', encoding='utf-8') as file:
        file.write(page_source)

    product_name = soup.find('div', attrs={"data-widget": 'webProductHeading'}).find(
        'h1').text.strip().replace('\t', '').replace('\n', ' ')
    image_url = ''
    try:
        img_tag = soup.find('img')
        if img_tag:
            image_url = img_tag.get('src') or ''
            if not image_url:
                srcset = img_tag.get('srcset') or ''
                if srcset:
                    # берём первую ссылку из srcset
                    image_url = srcset.split()[0]
    except:
        image_url = ''
    # product_id
    # try:
    #     product_id = soup.find('div', string=re.compile(
    #         'Артикул:')).text.split('Артикул: ')[1].strip()
    # except:
    #     product_id = None

    # product statistic
    try:
        product_statistic = soup.find(
            'div', attrs={"data-widget": 'webSingleProductScore'}).text.strip()

        if " • " in product_statistic:
            product_stars = product_statistic.split(' • ')[0].strip()
            product_reviews = product_statistic.split(' • ')[1].strip()
        else:
            product_statistic = product_statistic
    except:
        product_statistic = None
        product_stars = None
        product_reviews = None

    # product price
    try:
        price_element = soup.find(
            'span', string="без Ozon Карты").parent.parent.find('div').findAll('span')

        product_discount_price = price_element[0].text.strip(
        ) if price_element[0] else ''
        product_base_price = price_element[1].text.strip(
        ) if price_element[1] is not None else ''
    except:
        product_discount_price = None
        product_base_price = None

    # product price
    try:
        ozon_card_price_element = soup.find(
            'span', string="c Ozon Картой").parent.find('div').find('span')
    except AttributeError:
        card_price_div = soup.find(
            'div', attrs={"data-widget": "webPrice"}).findAll('span')

        product_base_price = card_price_div[0].text.strip()
        product_discount_price = card_price_div[1].text.strip()

    product_data = (
    {
        'product_url': url,
        'image_url': image_url,
        'product_id': product_id,
        'product_name': product_name,
        'product_discount_price': product_discount_price,
        'product_base_price': product_base_price,
        'product_statistic': product_statistic,
        'product_stars': product_stars,
        'product_reviews': product_reviews,
    }
)

    driver.close()
    driver.switch_to.window(driver.window_handles[0])

    return product_data


def get_products_links(item_name):
    options = uc.ChromeOptions()
    options.add_argument("--disable-blink-features=AutomationControlled")
    options.add_argument("--start-maximized")
    driver = uc.Chrome(options=options)
    driver.implicitly_wait(5)
    link = 'https://ozon.ru/search/?text=' + quote(item_name) + '&sorting=price'
    driver.get(url=link)
    wait = WebDriverWait(driver, 15)
    wait.until(EC.presence_of_element_located((By.CLASS_NAME, "tile-clickable-element")))

    try:
        find_links = driver.find_elements(By.CLASS_NAME, 'tile-clickable-element')
        products_urls = list(set([f'{link.get_attribute("href")}' for link in find_links]))
        print('[+] Ссылки на товары собраны!')
    except Exception as e:
        print(f'[!] Ошибка сбора ссылок: {e}')
        return []

    products_data = []
    for url in products_urls:
        try:
            data = collect_product_info(driver=driver, url=url)
            print(f'[+] Собрал данные товара с id: {data.get("product_id")}')
            time.sleep(2)
            products_data.append(data)
        except Exception as e:
            print(f'[!] Ошибка обработки товара {url}: {e}')
            continue

    driver.quit()
    return products_data

    
def main():
    

    query = "macbook"

    try:
        products_data = get_products_links(query)
        # GO вывод
        print(json.dumps(products_data, ensure_ascii=False, indent=2))
    except Exception as e:
        print(json.dumps({"error": str(e), "query": query}, ensure_ascii=False))
        sys.exit(2)

if __name__ == '__main__':
    main()