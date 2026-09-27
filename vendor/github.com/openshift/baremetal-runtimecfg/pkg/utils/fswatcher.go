package utils

import (
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

func CreateFileWatcher(log logrus.FieldLogger, fileName string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Error("Failed to add a create a new watcher")
		return nil, err
	}

	err = watcher.Add(fileName)
	if err != nil {
		log.WithFields(logrus.Fields{
			"filename": fileName,
		}).WithError(err).Error("Failed to add a watcher to file")
		return nil, err
	}

	return watcher, nil
}

func RunWatcher(log logrus.FieldLogger, watcher *fsnotify.Watcher, fileName string) (bool, error) {
	select {
	case event, ok := <-watcher.Events:
		if !ok {
			return false, nil
		}

		if event.Op&fsnotify.Write == fsnotify.Write {
			if stat, err := os.Stat(fileName); err != nil {
				log.WithFields(logrus.Fields{
					"filename": fileName,
				}).WithError(err).Error("Failed to stat file")
				return true, err
			} else if stat.Size() == 0 {
				// The file has been modified and truncated
				return false, nil
			}

			return true, nil
		}
	case err, ok := <-watcher.Errors:
		if !ok {
			return false, nil
		}

		log.WithFields(logrus.Fields{
			"filename": fileName,
		}).WithError(err).Error("File watcher error")
		return true, err
	}

	return false, nil
}
