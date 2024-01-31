package reconciler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-co-op/gocron"

	"github.com/k8snetworkplumbingwg/whereabouts/pkg/config"
	"github.com/k8snetworkplumbingwg/whereabouts/pkg/logging"
	"github.com/k8snetworkplumbingwg/whereabouts/pkg/types"
)

type ConfigWatcher struct {
	configDir       string
	configPath      string
	currentSchedule string
	scheduler       *gocron.Scheduler
	handlerFunc     func()
	watcher         *fsnotify.Watcher
}

func NewConfigWatcher(configPath string, scheduler *gocron.Scheduler, configWatcher *fsnotify.Watcher, handlerFunc func()) (*ConfigWatcher, error) {
	return newConfigWatcher(
		configPath,
		scheduler,
		configWatcher,
		handlerFunc,
	)
}

func newConfigWatcher(
	configPath string,
	scheduler *gocron.Scheduler,
	configWatcher *fsnotify.Watcher,
	handlerFunc func(),
) (*ConfigWatcher, error) {
	schedule, err := determineCronExpression(configPath)
	if err != nil {
		return nil, err
	}

	job, err := scheduler.Cron(schedule).Do(handlerFunc)
	if err != nil {
		return nil, fmt.Errorf("error creating job: %v", err)
	}

	_, err = scheduler.Job(job).Update()
	if err != nil {
		_ = logging.Errorf("error updating scheduler: %v", err)
	}

	return &ConfigWatcher{
		configDir:       filepath.Dir(configPath),
		configPath:      configPath,
		currentSchedule: schedule,
		scheduler:       scheduler,
		watcher:         configWatcher,
		handlerFunc:     handlerFunc,
	}, nil
}

func determineCronExpression(configPath string) (string, error) {
	// We read the expression from a file if present, otherwise we use ReconcilerCronExpression
	fileContents, err := os.ReadFile(configPath)
	if err != nil {
		flatipam, _, err := config.GetFlatIPAM(true, &types.IPAMConfig{}, "")
		if err != nil {
			return "", logging.Errorf("could not get flatipam config: %v", err)
		}

		_ = logging.Errorf("could not read file: %v, using expression from flatfile: %v", err, flatipam.IPAM.ReconcilerCronExpression)
		return flatipam.IPAM.ReconcilerCronExpression, nil
	}
	logging.Verbosef("using expression: %v", strings.TrimSpace(string(fileContents))) // do i need to trim spaces? idk i think the file would JUST be the expression?
	return strings.TrimSpace(string(fileContents)), nil
}

func (c *ConfigWatcher) SyncConfiguration(relevantEventPredicate func(event fsnotify.Event) bool) {
	go c.syncConfig(relevantEventPredicate)
	if err := c.watcher.Add(c.configDir); err != nil {
		_ = logging.Errorf("error adding watcher to config %q: %v", c.configPath, err)
	}
}

func (c *ConfigWatcher) syncConfig(relevantEventPredicate func(event fsnotify.Event) bool) {
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}

			if !relevantEventPredicate(event) {
				logging.Debugf("event not relevant: %v", event)
				continue
			}
			updatedSchedule, err := determineCronExpression(c.configPath)
			if err != nil {
				_ = logging.Errorf("error determining cron expression from %q: %v", c.configPath, err)
				continue
			}
			logging.Verbosef(
				"configuration updated to file %q. New cron expression: %s",
				event.Name,
				updatedSchedule,
			)

			if updatedSchedule == c.currentSchedule {
				logging.Debugf("no changes in schedule, nothing to do.")
				continue
			}
			job, err := c.scheduler.Cron(updatedSchedule).Do(c.handlerFunc)
			if err != nil {
				_ = logging.Errorf("error updating CRON configuration: %v", err)
				continue
			}

			_, err = c.scheduler.Job(job).Update()
			if err != nil {
				_ = logging.Errorf("error updating CRON configuration: %v", err)
				continue
			}
			c.currentSchedule = updatedSchedule
			logging.Verbosef(
				"successfully updated CRON configuration - new cron expression: %s",
				updatedSchedule,
			)
		case err, ok := <-c.watcher.Errors:
			_ = logging.Errorf("error when listening to config changes: %v", err)
			if !ok {
				return
			}
		}
	}
}
