package file

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ozonru/file.d/longpanic"
	"github.com/rjeczalik/notify"
	"go.uber.org/zap"
)

type watcher struct {
	path            string   // dir in which watch for files
	filenamePattern string   // files which match this pattern will be watched
	dirPattern      string   // dirs which match this pattern will be watched
	notifyFn        notifyFn // function to receive notifications
	watcherCh       chan notify.EventInfo
	logger          *zap.SugaredLogger
}

type notifyFn func(filename string, stat os.FileInfo)

// NewWatcher creates a watcher that see file creations in the path
// and if they match filePattern and dirPattern, pass them to notifyFn.
func NewWatcher(path string, filenamePattern string, dirPattern string, notifyFn notifyFn, logger *zap.SugaredLogger) *watcher {
	return &watcher{
		path:            path,
		filenamePattern: filenamePattern,
		dirPattern:      dirPattern,
		notifyFn:        notifyFn,
		logger:          logger,
	}
}

func (w *watcher) start() {
	w.logger.Infof("starting watcher path=%s, pattern=%s", w.path, w.filenamePattern)

	if _, err := filepath.Match(w.filenamePattern, "_"); err != nil {
		w.logger.Fatalf("wrong file name pattern %q: %s", w.filenamePattern, err.Error())
	}

	if _, err := filepath.Match(w.dirPattern, "_"); err != nil {
		w.logger.Fatalf("wrong dir name pattern %q: %s", w.dirPattern, err.Error())
	}

	eventsCh := make(chan notify.EventInfo, 1)
	w.watcherCh = eventsCh

	err := notify.Watch(w.path, eventsCh, notify.Create)
	if err != nil {
		w.logger.Warnf("can't create fs watcher: %s", err.Error())
		return
	}

	longpanic.Go(w.watch)

	w.tryAddPath(w.path)
}

func (w *watcher) stop() {
	w.logger.Infof("stopping watcher")

	notify.Stop(w.watcherCh)
	close(w.watcherCh)
}

func (w *watcher) tryAddPath(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}

	w.logger.Infof("starting path watch: %s ", path)

	for _, file := range files {
		if file.Name() == "" || file.Name() == "." || file.Name() == ".." {
			continue
		}

		w.notify(filepath.Join(path, file.Name()))
	}
}

func (w *watcher) notify(path string) {
	filename := path
	if filename == "" || filename == "." || filename == ".." {
		return
	}

	filename, err := filepath.Abs(filename)
	if err != nil {
		w.logger.Fatalf("can't get abs file name: %s", err.Error())
		return
	}

	stat, err := os.Lstat(filename)
	if err != nil {
		return
	}

	match, _ := filepath.Match(w.filenamePattern, filepath.Base(filename))
	if match {
		w.notifyFn(filename, stat)
	}

	match, _ = filepath.Match(w.dirPattern, filepath.Base(filename))
	if stat.IsDir() && match {
		w.tryAddPath(filename)
	}
}

func (w *watcher) watch() {
	for {
		event, ok := <-w.watcherCh
		if !ok {
			return
		}
		w.notify(event.Path())
	}
}
