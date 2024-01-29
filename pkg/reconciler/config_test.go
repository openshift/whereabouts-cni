package reconciler

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fsnotify/fsnotify"
	"github.com/go-co-op/gocron"
)

var _ = Describe("Reconciler configuration watcher", func() {
	var (
		config      *ConfigWatcher
		configDir   string
		dummyConfig *os.File
		mailbox     chan struct{}
		watcher     *fsnotify.Watcher
	)

	BeforeEach(func() {
		var err error

		mailbox = make(chan struct{})

		configDir, err = os.MkdirTemp("", "config")
		Expect(err).NotTo(HaveOccurred())
		const (
			initialCron   = "0/1 2 3 * *"
			dummyFileName = "DUMMY"
		)
		dummyConfig, err = os.Create(filepath.Join(configDir, filepath.Base(dummyFileName)))
		Expect(err).NotTo(HaveOccurred())

		Expect(dummyConfig.Write([]byte(initialCron))).To(Equal(len(initialCron)))
		scheduler := gocron.NewScheduler(time.UTC)
		watcher, err = fsnotify.NewWatcher()
		Expect(err).NotTo(HaveOccurred())
		config, err = newConfigWatcherForTests(
			dummyConfig.Name(),
			scheduler,
			watcher,
			func() { mailbox <- struct{}{} },
		)
		scheduler.StartAsync()
		Expect(err).NotTo(HaveOccurred())
		config.SyncConfiguration(func(event fsnotify.Event) bool {
			return event.Name == dummyConfig.Name() && event.Op&fsnotify.Write == fsnotify.Write
		})
	})

	AfterEach(func() {
		watcher.Close()
		dummyConfig.Close()
	})

	When("the cron job expression is updated in the file-system", func() {
		const updatedCron = "0/1 * * * *"

		BeforeEach(func() {
			Expect(dummyConfig.WriteAt([]byte(updatedCron), 0)).To(Equal(len(updatedCron)))
		})

		It("the current schedule is updated, and the handler function executed", func() {
			Eventually(func() string { return config.currentSchedule }).Should(Equal(updatedCron))
			Eventually(mailbox).WithTimeout(time.Minute).Should(Receive())
		})
	})
})

func newConfigWatcherForTests(configPath string, scheduler *gocron.Scheduler, configWatcher *fsnotify.Watcher, handlerFunc func()) (*ConfigWatcher, error) {
	return newConfigWatcher(
		configPath,
		scheduler,
		configWatcher,
		handlerFunc,
	)
}
