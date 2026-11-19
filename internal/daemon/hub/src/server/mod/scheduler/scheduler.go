package scheduler

import (
	"context"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/meta"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

const schedulerRefreshInterval = 5 * time.Second

type Scheduler struct {
	app.BaseModule

	CurrentApp     meta.CurrentApp        `inject:""`
	RegistryRepo   core.RegistryRepo      `inject:""`
	DescriptorRepo core.DescriptorRepo    `inject:""`
	NATSServer     *natsserver.NATSServer `inject:""`
	Flag           *hubflag.Flag          `inject:""`

	mutex     sync.Mutex
	cron      *cron.Cron
	jobs      map[_ScheduleConfig]cron.EntryID
	publisher _TaskPublisher
	stop      context.CancelFunc
	refreshWG sync.WaitGroup

	refreshInterval time.Duration
}

type _TaskPublisher interface {
	PublishTask(message taskspec.NATSMessage) error
}

// Replicas and descriptor versions declaring the same schedule share a timer.
type _ScheduleConfig struct {
	AppName      string
	TaskSkelName string
	core.TaskRunnerCronScheduler
}

func (s *Scheduler) DIInit() {
	s.cron = cron.New(cron.WithChain(cron.Recover(_SchedulerCronLogger{})))
	s.jobs = map[_ScheduleConfig]cron.EntryID{}
	if s.refreshInterval <= 0 {
		s.refreshInterval = schedulerRefreshInterval
	}
	if s.publisher == nil {
		s.publisher = &_NATSTaskPublisher{
			NATSServer: s.NATSServer,
			Flag:       s.Flag,
		}
	}
}

func (s *Scheduler) AfterAppStart() {
	s.refreshSchedulesSafely()
	s.cron.Start()

	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	s.refreshWG.Go(func() {
		s.refreshLoop(ctx)
	})
}

func (s *Scheduler) BeforeAppStop() {
	if s.stop != nil {
		s.stop()
	}
	s.refreshWG.Wait()
	if s.cron != nil {
		<-s.cron.Stop().Done()
	}
}

func (s *Scheduler) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshSchedulesSafely()
		}
	}
}

func (s *Scheduler) refreshSchedulesSafely() {
	defer func() {
		if recovered := recover(); recovered != nil {
			schedulerLogger.Error("scheduler schedule refresh panicked",
				"panic", recovered,
				"stack", string(debug.Stack()),
			)
		}
	}()
	if err := s.refreshSchedules(); err != nil {
		schedulerLogger.Error("scheduler schedule refresh failed", "error", err)
	}
}

func (s *Scheduler) refreshSchedules() error {
	configs, err := s.scheduleConfigs()
	if err != nil {
		return err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()

	addedJobs := map[_ScheduleConfig]cron.EntryID{}
	for config := range configs {
		if _, exists := s.jobs[config]; exists {
			continue
		}
		entryId, err := s.cron.AddFunc(config.CronExpr, func() {
			s.publishSchedule(config)
		})
		if err != nil {
			for _, addedEntryId := range addedJobs {
				s.cron.Remove(addedEntryId)
			}
			return fmt.Errorf("add cron schedule %+v: %w", config, err)
		}
		addedJobs[config] = entryId
	}
	maps.Copy(s.jobs, addedJobs)

	for config, entryId := range s.jobs {
		if _, exists := configs[config]; exists {
			continue
		}
		s.cron.Remove(entryId)
		delete(s.jobs, config)
	}
	return nil
}

func (s *Scheduler) scheduleConfigs() (map[_ScheduleConfig]struct{}, error) {
	configs := map[_ScheduleConfig]struct{}{}
	statuses := s.RegistryRepo.ListAppStatuses()
	versions := s.DescriptorRepo.ListTaskDescriptorVersions()
	for statusIndex, status := range statuses {
		if status == nil {
			return nil, fmt.Errorf("app status %d is nil", statusIndex)
		}
		for _, runner := range status.TaskRunners {
			for _, scheduler := range runner.CronSchedulers {
				config := _ScheduleConfig{
					AppName:                 status.Name,
					TaskSkelName:            runner.TaskSkelName,
					TaskRunnerCronScheduler: scheduler,
				}
				if err := config.validate(); err != nil {
					return nil, err
				}
				if err := config.checkNoArgumentTrigger(runner.DescriptorHash, versions); err != nil {
					return nil, err
				}
				configs[config] = struct{}{}
			}
		}
	}
	return configs, nil
}

func (config _ScheduleConfig) checkNoArgumentTrigger(descriptorHash string, versions []core.DescriptorVersion[*skeldesc.Task]) error {
	if descriptorHash == "" {
		return fmt.Errorf("scheduled task descriptor hash is empty")
	}
	for _, version := range versions {
		if version.Descriptor == nil {
			continue
		}
		if version.Descriptor.SkelName != config.TaskSkelName || version.DescriptorHash != descriptorHash {
			continue
		}
		for _, trigger := range version.Descriptor.Triggers {
			if trigger == nil {
				continue
			}
			if trigger.SkelName == config.TriggerSkelName {
				if len(trigger.Arguments) != 0 {
					return fmt.Errorf("scheduled task trigger %s must have no arguments", config.TriggerSkelName)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("scheduled task trigger %s not found on task %s", config.TriggerSkelName, config.TaskSkelName)
}

func (s *Scheduler) publishSchedule(config _ScheduleConfig) {
	if !s.hasActiveRunner(config) {
		return
	}

	trace := meta.InitialTrace()
	message := taskspec.NATSMessage{
		Metadata: taskspec.NATSMessageMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       s.CurrentApp.Name(),
			AppVersion:    s.CurrentApp.Version(),
			AppInstanceId: s.CurrentApp.InstanceId(),
			LaunchedAt:    skeltype.NewTimestampNow(),
		},
		TaskSkelName:    config.TaskSkelName,
		TriggerSkelName: config.TriggerSkelName,
		ArgumentsJson:   "{}",
	}
	if err := s.publisher.PublishTask(message); err != nil {
		schedulerLogger.Error("scheduler task publish failed",
			"appName", config.AppName,
			"taskSkelName", config.TaskSkelName,
			"triggerSkelName", config.TriggerSkelName,
			"error", err,
		)
	}
}

func (s *Scheduler) hasActiveRunner(config _ScheduleConfig) bool {
	statuses := s.RegistryRepo.ListAppStatuses()
	versions := s.DescriptorRepo.ListTaskDescriptorVersions()
	for _, status := range statuses {
		if status == nil || status.Name != config.AppName {
			continue
		}
		for _, runner := range status.TaskRunners {
			if runner.TaskSkelName != config.TaskSkelName || !slices.Contains(runner.CronSchedulers, config.TaskRunnerCronScheduler) {
				continue
			}
			if config.checkNoArgumentTrigger(runner.DescriptorHash, versions) == nil {
				return true
			}
		}
	}
	return false
}

func (c _ScheduleConfig) validate() error {
	if c.AppName == "" {
		return fmt.Errorf("scheduled task app name is empty")
	}
	if c.TaskSkelName == "" {
		return fmt.Errorf("scheduled task skel name is empty")
	}
	if c.TriggerSkelName == "" {
		return fmt.Errorf("scheduled task trigger skel name is empty")
	}
	if _, err := cron.ParseStandard(c.CronExpr); err != nil {
		return fmt.Errorf("scheduled task cron expression %q is invalid: %w", c.CronExpr, err)
	}
	return nil
}
