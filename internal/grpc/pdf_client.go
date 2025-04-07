package grpc

import (
	"astro-sarafan/internal/telegram"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"astro-sarafan/internal/database"
	_ "astro-sarafan/internal/models"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	// Импортируйте ваш proto-пакет
	gen_v1 "astro-sarafan/internal/pkg/gen_v1"
)

// PDFClient представляет клиент для взаимодействия с gRPC сервисом генерации PDF
type PDFClient struct {
	logger         *zap.Logger
	orderRepo      *database.OrderRepository
	telegram       *telegram.TelegramClient
	client         gen_v1.GeneratorV1Client
	conn           *grpc.ClientConn
	apiURL         string // URL для интерактивных кнопок в PDF
	pdfStoragePath string // Путь для сохранения PDF файлов
}

// NewPDFClient создает новый gRPC клиент для генерации PDF
func NewPDFClient(
	logger *zap.Logger,
	orderRepo *database.OrderRepository,
	grpcServerAddr string,
	apiURL string,
	pdfStoragePath string,
	telegram *telegram.TelegramClient,
) (*PDFClient, error) {
	// Устанавливаем gRPC соединение
	conn, err := grpc.Dial(grpcServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к gRPC серверу: %v", err)
	}

	// Создаем gRPC клиент
	client := gen_v1.NewGeneratorV1Client(conn)

	// Создаем директорию для хранения PDF, если её нет
	if err := os.MkdirAll(pdfStoragePath, 0755); err != nil {
		return nil, fmt.Errorf("ошибка создания директории для PDF: %v", err)
	}

	return &PDFClient{
		logger:         logger,
		orderRepo:      orderRepo,
		client:         client,
		conn:           conn,
		apiURL:         apiURL,
		pdfStoragePath: pdfStoragePath,
		telegram:       telegram,
	}, nil
}

// GenerateOrderPDF генерирует PDF для заказа
func (c *PDFClient) GenerateOrderPDF(orderID string) error {
	// Получаем информацию о заказе
	order, err := c.orderRepo.GetOrderByID(orderID)
	if err != nil {
		c.logger.Error("Ошибка при получении заказа",
			zap.Error(err),
			zap.String("order_id", orderID))
		return err
	}

	if order.ID == "" {
		return fmt.Errorf("заказ не найден: %s", orderID)
	}

	// Формируем запрос для gRPC сервиса
	req := &gen_v1.GenerateConsRequest{
		Name:      order.ClientName,
		UserId:    fmt.Sprintf("%d", order.ClientID),
		BirthDate: "", // Эти поля нужно заполнить, если они доступны
		BirthTime: "",
		BirthCity: &gen_v1.BirthCityInfo{
			City:        "",
			Coordinates: "",
		},
		KnowBirthTime:         true,
		SelectedConsultations: "consultation", // Укажите тип консультации
		ProductId:             orderID,        // Важно! Здесь передаем ID заказа как productId
	}

	// Устанавливаем таймаут для запроса
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Выполняем запрос к gRPC сервису
	resp, err := c.client.GenerateCons(ctx, req)
	if err != nil {
		c.logger.Error("Ошибка при генерации PDF через gRPC",
			zap.Error(err),
			zap.String("order_id", orderID))
		return err
	}

	// Сохраняем полученный PDF
	pdfBytes := resp.GetCons()
	pdfFilename := fmt.Sprintf("%s/%s.pdf", c.pdfStoragePath, orderID)

	if err := ioutil.WriteFile(pdfFilename, pdfBytes, 0644); err != nil {
		c.logger.Error("Ошибка при сохранении PDF файла",
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("filename", pdfFilename))
		return err
	}

	// Формируем URL для доступа к PDF
	pdfURL := fmt.Sprintf("%s/pdf/%s.pdf", c.apiURL, orderID)

	// Обновляем информацию о PDF в базе данных
	if err := c.orderRepo.UpdatePDFInfo(orderID, pdfURL); err != nil {
		c.logger.Error("Ошибка при обновлении информации о PDF в базе данных",
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("pdf_url", pdfURL))
		return err
	}

	// Отправляем сообщение клиенту о готовности консультации
	clientID := order.ClientID
	messageText := fmt.Sprintf(
		"✨ Ваша астрологическая консультация готова! Вы можете ознакомиться с ней по ссылке: %s",
		pdfURL)

	if err := c.telegram.SendMessage(clientID, messageText); err != nil {
		c.logger.Error("Ошибка при отправке сообщения клиенту",
			zap.Error(err),
			zap.Int64("client_id", clientID))
		// Не возвращаем ошибку, так как PDF уже сгенерирован
	}

	c.logger.Info("PDF успешно сгенерирован и сохранен",
		zap.String("order_id", orderID),
		zap.String("pdf_url", pdfURL),
		zap.Int64("client_id", clientID))

	return nil
}

// CheckAndGeneratePDFs проверяет заказы и генерирует PDF для тех, которые в работе и без PDF
func (c *PDFClient) CheckAndGeneratePDFs() {
	c.logger.Info("Проверка заказов для генерации PDF")

	// Получаем заказы, которым нужно сгенерировать PDF
	orders, err := c.orderRepo.GetOrdersInWorkWithoutPDF()
	if err != nil {
		c.logger.Error("Ошибка при получении заказов без PDF",
			zap.Error(err))
		return
	}

	if len(orders) == 0 {
		c.logger.Debug("Нет заказов, требующих генерации PDF")
		return
	}

	c.logger.Info("Найдены заказы для генерации PDF",
		zap.Int("count", len(orders)))

	// Генерируем PDF для каждого заказа
	for _, order := range orders {
		if err := c.GenerateOrderPDF(order.ID); err != nil {
			c.logger.Error("Ошибка при генерации PDF для заказа",
				zap.Error(err),
				zap.String("order_id", order.ID))
			continue
		}

		c.logger.Info("Успешно сгенерирован PDF для заказа",
			zap.String("order_id", order.ID))
	}
}

// StartCheckingLoop запускает периодическую проверку и генерацию PDF
func (c *PDFClient) StartCheckingLoop(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.CheckAndGeneratePDFs()
		}
	}()

	c.logger.Info("Запущен цикл проверки и генерации PDF",
		zap.Duration("interval", interval))
}

// Close закрывает gRPC соединение
func (c *PDFClient) Close() error {
	return c.conn.Close()
}
