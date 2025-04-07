package api

import (
	"astro-sarafan/internal/database"
	"html/template"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ButtonServer представляет HTTP-сервер для обработки нажатий на кнопку в PDF
type ButtonServer struct {
	logger     *zap.Logger
	orderRepo  *database.OrderRepository
	httpServer *http.Server
	templates  *template.Template
	pdfPath    string // Путь к директории с PDF файлами
}

// NewButtonServer создает новый сервер для обработки нажатий кнопки
func NewButtonServer(
	logger *zap.Logger,
	orderRepo *database.OrderRepository,
	addr string,
	pdfPath string,
) *ButtonServer {
	// Создаем шаблоны для страниц
	tmpl := template.Must(template.New("").Parse(`
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
            <meta name="viewport" content="width=device-width, initial-scale=1.0">
            <title>Астрологическая консультация</title>
            <style>
                body {
                    font-family: 'Arial', sans-serif;
                    line-height: 1.6;
                    color: #333;
                    max-width: 800px;
                    margin: 0 auto;
                    padding: 20px;
                    text-align: center;
                }
                .container {
                    background-color: #f9f9f9;
                    border-radius: 10px;
                    padding: 30px;
                    box-shadow: 0 0 10px rgba(0,0,0,0.1);
                }
                h1 {
                    color: #1a237e;
                    margin-bottom: 30px;
                }
                .success-icon {
                    font-size: 80px;
                    color: #4CAF50;
                    margin-bottom: 20px;
                }
                .message {
                    font-size: 18px;
                    margin-bottom: 30px;
                }
                .btn {
                    display: inline-block;
                    background-color: #1a237e;
                    color: white;
                    padding: 12px 24px;
                    text-decoration: none;
                    border-radius: 4px;
                    font-weight: bold;
                    transition: background-color 0.3s;
                }
                .btn:hover {
                    background-color: #3949ab;
                }
            </style>
        </head>
        <body>
            <div class="container">
                <h1>Астрологическая консультация</h1>
                <div class="success-icon">✓</div>
                <div class="message">
                    {{.Message}}
                </div>
                {{if .ShowButton}}
                <a href="{{.ButtonLink}}" class="btn">Перейти к оплате</a>
                {{end}}
            </div>
        </body>
        </html>
    `))

	// Создаем HTTP-сервер
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &ButtonServer{
		logger:     logger,
		orderRepo:  orderRepo,
		httpServer: server,
		templates:  tmpl,
		pdfPath:    pdfPath,
	}
}

// Start запускает HTTP-сервер
func (s *ButtonServer) Start() {
	// Регистрируем обработчики
	http.HandleFunc("/buy", s.handleBuyButton)
	http.HandleFunc("/payment", s.handlePayment)
	http.HandleFunc("/pdf/", s.handlePDFDownload)

	// Запускаем сервер в отдельной горутине
	go func() {
		s.logger.Info("Запуск HTTP-сервера", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Ошибка запуска HTTP-сервера", zap.Error(err))
		}
	}()
}

// Остановка сервера
func (s *ButtonServer) Stop() error {
	return s.httpServer.Close()
}

// handleBuyButton обрабатывает нажатие на кнопку "Купить консультацию"
func (s *ButtonServer) handleBuyButton(w http.ResponseWriter, r *http.Request) {
	// Получаем ID заказа из параметров запроса
	orderID := r.URL.Query().Get("order")
	if orderID == "" {
		s.renderErrorPage(w, "Не указан идентификатор заказа")
		return
	}

	// Получаем заказ по ID
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		s.logger.Error("Ошибка при получении заказа",
			zap.Error(err),
			zap.String("order_id", orderID))
		s.renderErrorPage(w, "Ошибка при получении информации о заказе")
		return
	}

	if order.ID == "" {
		s.renderErrorPage(w, "Заказ не найден")
		return
	}

	// Обновляем статус нажатия кнопки
	err = s.orderRepo.UpdateButtonPressed(orderID)
	if err != nil {
		s.logger.Error("Ошибка при обновлении статуса кнопки",
			zap.Error(err),
			zap.String("order_id", orderID))
		s.renderErrorPage(w, "Ошибка при обработке запроса")
		return
	}

	// Отправляем уведомление астрологу о том, что клиент нажал кнопку
	// Здесь должен быть код для отправки уведомления

	// Перенаправляем на страницу оплаты
	http.Redirect(w, r, "/payment?order="+orderID, http.StatusSeeOther)
}

// handlePayment обрабатывает страницу оплаты
func (s *ButtonServer) handlePayment(w http.ResponseWriter, r *http.Request) {
	// Получаем ID заказа из параметров запроса
	orderID := r.URL.Query().Get("order")
	if orderID == "" {
		s.renderErrorPage(w, "Не указан идентификатор заказа")
		return
	}

	// Получаем заказ по ID
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		s.logger.Error("Ошибка при получении заказа",
			zap.Error(err),
			zap.String("order_id", orderID))
		s.renderErrorPage(w, "Ошибка при получении информации о заказе")
		return
	}

	if order.ID == "" {
		s.renderErrorPage(w, "Заказ не найден")
		return
	}

	// Здесь должна быть интеграция с платежной системой
	// В данном примере просто отображаем страницу успешного перехода

	data := struct {
		Message    string
		ShowButton bool
		ButtonLink string
	}{
		Message:    "Спасибо за интерес к полной астрологической консультации! Мы свяжемся с вами в ближайшее время для уточнения деталей оплаты.",
		ShowButton: false,
	}

	s.templates.Execute(w, data)
}

// handlePDFDownload обрабатывает скачивание PDF-файлов
func (s *ButtonServer) handlePDFDownload(w http.ResponseWriter, r *http.Request) {
	// Извлекаем имя файла из URL
	filename := r.URL.Path[len("/pdf/"):]
	if filename == "" {
		http.Error(w, "Имя файла не указано", http.StatusBadRequest)
		return
	}

	// Отдаем файл
	http.ServeFile(w, r, s.pdfPath+"/"+filename)
}

// renderErrorPage отображает страницу с ошибкой
func (s *ButtonServer) renderErrorPage(w http.ResponseWriter, message string) {
	data := struct {
		Message    string
		ShowButton bool
		ButtonLink string
	}{
		Message:    "Произошла ошибка: " + message,
		ShowButton: false,
	}

	w.WriteHeader(http.StatusBadRequest)
	s.templates.Execute(w, data)
}
