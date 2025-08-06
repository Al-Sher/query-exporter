package scan_queries

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"time"
)

// QueryWatcher структура для отслеживания за изменениями файлов
type QueryWatcher struct {
	watcher    *fsnotify.Watcher
	folderPath string
	EventChan  chan struct{}
	debounce   time.Duration
	lastEvent  time.Time
}

// NewWatcher создать нового наблюдателя за файлами
func NewWatcher(path string, debounce time.Duration) (*QueryWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(path); err != nil {
		return nil, err
	}

	return &QueryWatcher{
		watcher:    watcher,
		folderPath: path,
		EventChan:  make(chan struct{}, 1),
		debounce:   debounce,
	}, nil
}

// Run запустить наблюдателя
func (qw *QueryWatcher) Run(logger *zap.Logger) {
	for {
		select {
		case event, ok := <-qw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove {
				qw.handleEvent()
			}
		case err, ok := <-qw.watcher.Errors:
			if !ok {
				return
			}
			logger.Error(fmt.Sprintf("ошибка наблюдателя: %v", err))
		}
	}
}

// handleEvent событие, запускаемое при изменении файлов
// нет учета какой именно файл изменился из-за необходимости поддерживать
// заддержки на сохранение файла в ФС после получения сигнала
func (qw *QueryWatcher) handleEvent() {
	now := time.Now()
	if now.Sub(qw.lastEvent) < qw.debounce {
		return
	}
	qw.lastEvent = now

	eventTime := time.Now()

	time.AfterFunc(500*time.Millisecond, func() {
		if time.Since(eventTime) >= 500*time.Millisecond {
			select {
			case qw.EventChan <- struct{}{}:
			default:
			}
		}
	})
}

// Close закрываем наблюдателя
func (qw *QueryWatcher) Close(logger *zap.Logger) {
	err := qw.watcher.Close()
	if err != nil {
		logger.Error(fmt.Sprintf("ошибка завершения наблюдателя: %v\n", err))

		return
	}
}
