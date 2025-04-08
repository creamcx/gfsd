package database

import (
	"astro-sarafan/internal/config"
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // драйвер для PostgreSQL
	"go.uber.org/zap"
)

// NewConnection создает новое подключение к базе данных
func NewConnection(cfg config.Database, logger *zap.Logger) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logger.Error("Ошибка подключения к базе данных", zap.Error(err))
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	// Установка настроек пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Проверка подключения
	if err := db.Ping(); err != nil {
		logger.Error("Ошибка проверки подключения к базе данных", zap.Error(err))
		return nil, fmt.Errorf("не удалось проверить подключение к базе данных: %w", err)
	}

	logger.Info("Успешное подключение к базе данных")
	return db, nil
}

// MigrateUp выполняет миграции базы данных
func MigrateUp(cfg config.Database, logger *zap.Logger, verbose bool) error {
	logger.Info("Запуск миграций базы данных")
	startTime := time.Now()

	// Проверяем путь к миграциям
	absPath, err := filepath.Abs(cfg.MigrationsPath)
	if err != nil {
		logger.Error("Не удалось получить абсолютный путь к директории миграций",
			zap.String("path", cfg.MigrationsPath),
			zap.Error(err),
		)
		return fmt.Errorf("не удалось получить абсолютный путь к миграциям: %w", err)
	}

	// Проверяем существование директории
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		logger.Error("Директория миграций не существует",
			zap.String("path", absPath),
		)
		return fmt.Errorf("директория миграций не существует: %s", absPath)
	}

	// Проверяем наличие файлов миграций
	files, err := os.ReadDir(absPath)
	if err != nil {
		logger.Error("Ошибка чтения директории миграций",
			zap.String("path", absPath),
			zap.Error(err),
		)
		return fmt.Errorf("ошибка чтения директории миграций: %w", err)
	}

	if len(files) == 0 {
		logger.Error("Директория миграций пуста",
			zap.String("path", absPath),
		)
		return fmt.Errorf("директория миграций пуста: %s", absPath)
	}

	if verbose {
		logger.Info("Найдены файлы миграций",
			zap.String("path", absPath),
			zap.Int("file_count", len(files)),
		)

		for i, file := range files {
			logger.Info(fmt.Sprintf("Файл миграции #%d", i+1),
				zap.String("name", file.Name()),
				zap.Bool("is_dir", file.IsDir()),
			)
		}
	}

	// Формируем строку подключения
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	// Логируем параметры подключения (без пароля)
	logger.Info("Параметры подключения к БД",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("user", cfg.User),
		zap.String("database", cfg.DBName),
		zap.String("sslmode", cfg.SSLMode),
	)

	// Пытаемся подключиться к базе данных напрямую для проверки соединения
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("Ошибка подключения к базе данных", zap.Error(err))
		return fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	if err := db.Ping(); err != nil {
		logger.Error("Ошибка проверки соединения с базой данных", zap.Error(err))
		return fmt.Errorf("ошибка проверки соединения с базой данных: %w", err)
	}

	logger.Info("Соединение с базой данных успешно установлено")

	// Создаем драйвер для migrate
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Error("Ошибка создания драйвера миграций", zap.Error(err))
		return fmt.Errorf("ошибка создания драйвера миграций: %w", err)
	}

	// Создаем экземпляр migrate с нашим драйвером и путем к файлам миграций
	sourceURL := fmt.Sprintf("file://%s", absPath)
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		logger.Error("Ошибка создания инстанса мигратора",
			zap.String("source_url", sourceURL),
			zap.Error(err),
		)
		return fmt.Errorf("ошибка создания инстанса мигратора: %w", err)
	}

	// Получаем текущую версию миграции перед применением новых
	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		logger.Error("Ошибка получения текущей версии миграций", zap.Error(err))
		return fmt.Errorf("ошибка получения текущей версии миграций: %w", err)
	}

	if errors.Is(err, migrate.ErrNilVersion) {
		logger.Info("Текущая версия миграций: не инициализирована")
	} else {
		logger.Info("Текущая версия миграций",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	// Выполняем миграцию
	logger.Info("Начало выполнения миграций")
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("Ошибка выполнения миграций", zap.Error(err))
		return fmt.Errorf("ошибка выполнения миграций: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("Миграции не требуются: база данных уже в актуальном состоянии")
	} else {
		// Получаем новую версию после применения миграций
		newVersion, dirty, err := m.Version()
		if err != nil {
			logger.Warn("Ошибка получения новой версии миграций", zap.Error(err))
		} else {
			logger.Info("Новая версия миграций",
				zap.Uint("version", newVersion),
				zap.Bool("dirty", dirty),
			)
		}
	}

	duration := time.Since(startTime)
	logger.Info("Миграции успешно выполнены",
		zap.Duration("duration", duration),
	)
	return nil
}

// MigrateDown откатывает миграции базы данных
func MigrateDown(cfg config.Database, logger *zap.Logger, verbose bool) error {
	logger.Info("Запуск отката миграций базы данных")
	startTime := time.Now()

	// Проверяем путь к миграциям
	absPath, err := filepath.Abs(cfg.MigrationsPath)
	if err != nil {
		logger.Error("Не удалось получить абсолютный путь к директории миграций",
			zap.String("path", cfg.MigrationsPath),
			zap.Error(err),
		)
		return fmt.Errorf("не удалось получить абсолютный путь к миграциям: %w", err)
	}

	// Формируем строку подключения
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	// Логируем параметры подключения (без пароля)
	logger.Info("Параметры подключения к БД",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("user", cfg.User),
		zap.String("database", cfg.DBName),
		zap.String("sslmode", cfg.SSLMode),
	)

	// Пытаемся подключиться к базе данных напрямую для проверки соединения
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("Ошибка подключения к базе данных", zap.Error(err))
		return fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	if err := db.Ping(); err != nil {
		logger.Error("Ошибка проверки соединения с базой данных", zap.Error(err))
		return fmt.Errorf("ошибка проверки соединения с базой данных: %w", err)
	}

	logger.Info("Соединение с базой данных успешно установлено")

	// Создаем драйвер для migrate
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Error("Ошибка создания драйвера миграций", zap.Error(err))
		return fmt.Errorf("ошибка создания драйвера миграций: %w", err)
	}

	// Создаем экземпляр migrate с нашим драйвером и путем к файлам миграций
	sourceURL := fmt.Sprintf("file://%s", absPath)
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		logger.Error("Ошибка создания инстанса мигратора",
			zap.String("source_url", sourceURL),
			zap.Error(err),
		)
		return fmt.Errorf("ошибка создания инстанса мигратора: %w", err)
	}

	// Получаем текущую версию миграции перед откатом
	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		logger.Error("Ошибка получения текущей версии миграций", zap.Error(err))
		return fmt.Errorf("ошибка получения текущей версии миграций: %w", err)
	}

	if errors.Is(err, migrate.ErrNilVersion) {
		logger.Info("Текущая версия миграций: не инициализирована")
		logger.Warn("Нет миграций для отката")
		return nil
	} else {
		logger.Info("Текущая версия миграций перед откатом",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	// Выполняем откат миграции
	logger.Info("Начало отката миграций")

	if verbose {
		logger.Info("Запрашивается откат к предыдущей версии")
	}

	if err = m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("Ошибка отката миграций", zap.Error(err))
		return fmt.Errorf("ошибка отката миграций: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("Откат не требуется: база данных уже в исходном состоянии")
	} else {
		// Получаем новую версию после отката миграций
		newVersion, dirty, err := m.Version()
		if err != nil {
			logger.Warn("Ошибка получения новой версии миграций после отката", zap.Error(err))
		} else {
			logger.Info("Новая версия миграций после отката",
				zap.Uint("version", newVersion),
				zap.Bool("dirty", dirty),
			)
		}
	}

	duration := time.Since(startTime)
	logger.Info("Откат миграций успешно выполнен",
		zap.Duration("duration", duration),
	)
	return nil
}
