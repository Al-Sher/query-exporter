package scan_queries

import (
	"fmt"
	"github.com/Al-Sher/query-exporter/internal/validator"
	"os"
	"path/filepath"
	"strings"
)

// Query структура sql запроса
type Query struct {
	Sql  string
	Name string
}

// Queries структура набора sql запрсоов
type Queries map[string]Query

// Scan сканирование папки с файлами для сборки sql запросов
func Scan(path string, qv validator.QueryValidator) (Queries, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("не существует файл или директория: %s", path)
	}

	files, err := os.ReadDir(path)

	if err != nil {
		return nil, err
	}

	var queries = make(Queries)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		metricName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		filePath := filepath.Join(path, file.Name())
		contents, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		contentStr := string(contents)
		if err := qv.ValidateQuery(contentStr); err != nil {
			return nil, err
		}

		q := Query{
			Sql:  contentStr,
			Name: metricName,
		}

		queries[metricName] = q
	}

	return queries, nil
}
