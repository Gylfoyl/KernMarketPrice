package models

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
