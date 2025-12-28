import sys
import json
from functions import page_down, collect_product_info, get_products_links


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "missing query"}))
        sys.exit(1)

    query = sys.argv[1]

    try:
        products_data = get_products_links(query)
        print(json.dumps({"source": "python_fallback", "query": query, "items": products_data}, ensure_ascii=False))
    except Exception as e:
        print(json.dumps({"error": str(e), "query": query}, ensure_ascii=False))
        sys.exit(2)


if __name__ == '__main__':
    main()