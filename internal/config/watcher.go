package config

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = ctrl.Log.WithName("configWatcher")
)

func WatchConfigFile(configFilePath string, cm *ConfigManager) {
	dir := filepath.Dir(configFilePath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err, "failed to create fsnotify watcher")
	}

	defer func() {
		if err := watcher.Close(); err != nil {
			log.Error(err, "failed to close watcher")
		}
	}()

	if err := watcher.Add(dir); err != nil {
		log.Error(err, "failed to watch directory %s", "directory", dir)
	}

	log.Info("Watching config dir for changes: %s", "directory", dir)

	var debounceTimer *time.Timer
	var timerMu sync.Mutex

	triggerReload := func() {
		if err := cm.LoadFromFile(configFilePath); err != nil {
			log.Error(err, "failed to reload config")
		} else {
			log.Info("config reloaded successfully")
		}
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Kubernetes updates the file via atomic rename
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod) != 0 {
				if filepath.Base(event.Name) == filepath.Base(configFilePath) {
					log.Info("Config file changed: %s", "event", event)

					timerMu.Lock()
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
						triggerReload()
					})
					timerMu.Unlock()
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error(err, "fsnotify error")
		}
	}
}
