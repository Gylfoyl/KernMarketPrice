import React, { useState } from 'react';
const API_ENDPOINT = 'BACKEND_ENDPOINT';

export default function App() {
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState(null);

  async function handleSearch(e) {
    e.preventDefault();
    if (!query.trim()) return;

    setLoading(true);
    setError('');
    setResult(null);

    try {
      const res = await fetch(`${API_ENDPOINT}?q=${encodeURIComponent(query.trim())}`);
      if (!res.ok) throw new Error('Ошибка при запросе к серверу');

      const data = await res.json();
      setResult(data);
    } catch (err) {
      setError(err.message || 'Произошла ошибка. Попробуйте позже.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="app">
      <main className="app-main">
        <div className="container">
          <h1 className="title">Поиск самой низкой цены</h1>
          <p className="subtitle">Введите название товара — покажем самое выгодное предложение из Ozon и Wildberries</p>

          <form onSubmit={handleSearch} className="search-form">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Например: iPhone 13"
              className="search-input"
            />
            <button type="submit" className="search-button" disabled={loading}>
              {loading ? 'Поиск...' : 'Найти'}
            </button>
          </form>

          {error && <div className="error-box">{error}</div>}

          {result && (
            <div className="result-wrapper">
              <div className="result-card">
                {result.image && (
                  <img src={result.image} alt={result.title} className="result-image" />
                )}
                <div className="result-info">
                  <h2 className="result-title">{result.title}</h2>
                  <p className="result-market">Маркетплейс: <span>{result.marketplace}</span></p>
                  <p className="result-price">{result.price?.toLocaleString()} ₽</p>
                  {result.link && (
                    <a
                      href={result.link}
                      target="_blank"
                      rel="noreferrer"
                      className="result-link"
                    >
                      Перейти к товару
                    </a>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </main>

      <footer className="footer">
        <div className="footer-inner">
          <p>© {new Date().getFullYear()} Поиск выгодных цен</p>
          <a href="https://t.me/your_telegram" target="_blank" rel="noreferrer" className="footer-link">
            Мой Telegram
          </a>
        </div>
      </footer>
    </div>
  );
}
