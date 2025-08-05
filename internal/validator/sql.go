package validator

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xwb1989/sqlparser"
	"regexp"
	"strings"
)

var (
	validMetricName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	validLabelName  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

type QueryValidator struct {
	invalidQueries prometheus.Counter
}

func NewQueryValidator(namespace string) *QueryValidator {
	return &QueryValidator{
		invalidQueries: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "invalid_queries_total",
			Help:      "Total number of invalid SQL queries detected",
		}),
	}
}

func (qv *QueryValidator) Describe(ch chan<- *prometheus.Desc) {
	qv.invalidQueries.Describe(ch)
}

func (qv *QueryValidator) Collect(ch chan<- prometheus.Metric) {
	qv.invalidQueries.Collect(ch)
}

func (qv *QueryValidator) ValidateQuery(query string) error {
	if strings.TrimSpace(query) == "" {
		qv.invalidQueries.Inc()
		return fmt.Errorf("пустой запрос")
	}

	_, err := sqlparser.Parse(query)
	if err != nil {
		qv.invalidQueries.Inc()
		return fmt.Errorf("ошибка sql синтаксиса: %v", err)
	}

	stmt, err := sqlparser.Parse(query)
	if err == nil {
		switch stmt.(type) {
		case *sqlparser.Select, *sqlparser.Show, *sqlparser.OtherRead:
			return nil
		default:
			qv.invalidQueries.Inc()
			return fmt.Errorf("доступны только запросы типа SELECT/SHOW")
		}
	}

	return nil
}

func (qv *QueryValidator) ValidateMetricName(name string) error {
	if !validMetricName.MatchString(name) {
		return fmt.Errorf("невалидное название метрики '%s'. не соответствует регулярке %s", name, validMetricName.String())
	}
	return nil
}

func (qv *QueryValidator) ValidateLabelNames(labels []string) error {
	for _, label := range labels {
		if !validLabelName.MatchString(label) {
			return fmt.Errorf("невалидное название label '%s'. не соответствует регулярке %s", label, validLabelName.String())
		}
	}
	return nil
}
