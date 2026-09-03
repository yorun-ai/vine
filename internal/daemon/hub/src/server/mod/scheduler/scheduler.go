package scheduler

import (
	"context"
	"fmt"
	"maps"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/robfig/cron/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

const schedulerRefreshInterval = 5 * time.Second

const (
	schedulerClientName       = "vine.hub.scheduler"
	schedulerClientVersion    = "0.0.0"
	schedulerClientInstanceId = "00000000-0000-0000-0000-000000000000"
)

type Scheduler struct {
	app.BaseModule

	RegistryRepo core.RegistryRepo      `inject:""`
	SchemaRepo   core.SchemaRepo        `inject:""`
	NATSServer   *natsserver.NATSServer `inject:""`
	Flag         *hubflag.Flag          `inject:""`

	mutex     sync.Mutex
	cron      *cron.Cron
	jobs      map[string]cron.EntryID
	publisher _TaskPublisher
	stop      context.CancelFunc
	refreshWG sync.WaitGroup

	refreshInterval time.Duration
}

type _TaskPublisher interface {
	PublishTask(message taskspec.NATSMessage) error
}

type _ScheduleConfig struct {
	AppName         string
	TaskSkelName    string
	SchemaHash      string
	TriggerSkelName string
	CronExpr        string
}

func (s *Scheduler) DIInit() {
	s.cron = cron.New(cron.WithChain(cron.Recover(_SchedulerCronLogger{})))
	s.jobs = map[string]cron.EntryID{}
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
	nextKeys := map[string]struct{}{}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	addedJobs := map[string]cron.EntryID{}
	for _, config := range configs {
		key := config.key()
		nextKeys[key] = struct{}{}
		if _, exists := s.jobs[key]; exists {
			continue
		}
		addedConfig := config
		entryId, err := s.cron.AddFunc(config.CronExpr, func() {
			s.publishSchedule(addedConfig)
		})
		if err != nil {
			for _, addedEntryId := range addedJobs {
				s.cron.Remove(addedEntryId)
			}
			return fmt.Errorf("add cron schedule %q: %w", key, err)
		}
		addedJobs[key] = entryId
	}
	maps.Copy(s.jobs, addedJobs)

	for key, entryId := range s.jobs {
		if _, exists := nextKeys[key]; exists {
			continue
		}
		s.cron.Remove(entryId)
		delete(s.jobs, key)
	}
	return nil
}

func (s *Scheduler) scheduleConfigs() ([]_ScheduleConfig, error) {
	configs := []_ScheduleConfig{}
	seen := map[string]struct{}{}
	for statusIndex, status := range s.RegistryRepo.ListAppStatuses() {
		if status == nil {
			return nil, fmt.Errorf("app status %d is nil", statusIndex)
		}
		for _, runner := range status.TaskRunners {
			for _, scheduler := range runner.CronSchedulers {
				config := _ScheduleConfig{
					AppName:         status.Name,
					TaskSkelName:    runner.TaskSkelName,
					SchemaHash:      runner.SchemaHash,
					TriggerSkelName: scheduler.TriggerSkelName,
					CronExpr:        scheduler.CronExpr,
				}
				if err := config.validate(); err != nil {
					return nil, err
				}
				if err := s.checkNoArgumentTrigger(config); err != nil {
					return nil, err
				}
				key := config.key()
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				configs = append(configs, config)
			}
		}
	}
	return configs, nil
}

func (s *Scheduler) checkNoArgumentTrigger(config _ScheduleConfig) error {
	for _, version := range s.SchemaRepo.ListTaskSchemaVersions() {
		if version.Schema == nil {
			continue
		}
		if version.Schema.SkelName != config.TaskSkelName || version.SchemaHash != config.SchemaHash {
			continue
		}
		for _, trigger := range version.Schema.Triggers {
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
			AppName:       schedulerClientName,
			AppVersion:    schedulerClientVersion,
			AppInstanceId: skel.NewUUID(uuid.MustParse(schedulerClientInstanceId)),
			LaunchedAt:    skel.NewTimestampNow(),
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
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		if status == nil {
			continue
		}
		if status.Name != config.AppName {
			continue
		}
		for _, runner := range status.TaskRunners {
			if runner.TaskSkelName != config.TaskSkelName || runner.SchemaHash != config.SchemaHash {
				continue
			}
			for _, scheduler := range runner.CronSchedulers {
				if scheduler.TriggerSkelName == config.TriggerSkelName && scheduler.CronExpr == config.CronExpr {
					return true
				}
			}
		}
	}
	return false
}

func (c _ScheduleConfig) key() string {
	return strings.Join([]string{c.AppName, c.TaskSkelName, c.SchemaHash, c.TriggerSkelName, c.CronExpr}, "\x00")
}

func (c _ScheduleConfig) validate() error {
	if c.AppName == "" {
		return fmt.Errorf("scheduled task app name is empty")
	}
	if c.TaskSkelName == "" {
		return fmt.Errorf("scheduled task skel name is empty")
	}
	if c.SchemaHash == "" {
		return fmt.Errorf("scheduled task schema hash is empty")
	}
	if c.TriggerSkelName == "" {
		return fmt.Errorf("scheduled task trigger skel name is empty")
	}
	if _, err := cron.ParseStandard(c.CronExpr); err != nil {
		return fmt.Errorf("scheduled task cron expression %q is invalid: %w", c.CronExpr, err)
	}
	return nil
}
