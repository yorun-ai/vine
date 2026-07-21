package scheduler

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
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
}

type _TaskPublisher interface {
	PublishTask(message taskspec.NATSMessage)
}

type _ScheduleConfig struct {
	AppName         string
	TaskSkelName    string
	SchemaHash      string
	TriggerSkelName string
	CronExpr        string
}

func (s *Scheduler) DIInit() {
	s.cron = cron.New()
	s.jobs = map[string]cron.EntryID{}
	if s.publisher == nil {
		s.publisher = &_NATSTaskPublisher{
			NATSServer: s.NATSServer,
			Flag:       s.Flag,
		}
	}
}

func (s *Scheduler) AfterAppStart() {
	s.refreshSchedules()
	s.cron.Start()

	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	go s.refreshLoop(ctx)
}

func (s *Scheduler) BeforeAppStop() {
	if s.stop != nil {
		s.stop()
	}
	if s.cron != nil {
		s.cron.Stop()
	}
}

func (s *Scheduler) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(schedulerRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshSchedules()
		}
	}
}

func (s *Scheduler) refreshSchedules() {
	configs := s.scheduleConfigs()
	nextKeys := map[string]struct{}{}

	s.mutex.Lock()
	defer s.mutex.Unlock()

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
		ex.PanicIfError(err)
		s.jobs[key] = entryId
	}

	for key, entryId := range s.jobs {
		if _, exists := nextKeys[key]; exists {
			continue
		}
		s.cron.Remove(entryId)
		delete(s.jobs, key)
	}
}

func (s *Scheduler) scheduleConfigs() []_ScheduleConfig {
	configs := []_ScheduleConfig{}
	seen := map[string]struct{}{}
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		for _, runner := range status.TaskRunners {
			for _, scheduler := range runner.CronSchedulers {
				config := _ScheduleConfig{
					AppName:         status.Name,
					TaskSkelName:    runner.TaskSkelName,
					SchemaHash:      runner.SchemaHash,
					TriggerSkelName: scheduler.TriggerSkelName,
					CronExpr:        scheduler.CronExpr,
				}
				s.checkNoArgumentTrigger(config)
				key := config.key()
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				configs = append(configs, config)
			}
		}
	}
	return configs
}

func (s *Scheduler) checkNoArgumentTrigger(config _ScheduleConfig) {
	for _, version := range s.SchemaRepo.ListTaskSchemaVersions() {
		if version.Schema.SkelName != config.TaskSkelName || version.SchemaHash != config.SchemaHash {
			continue
		}
		for _, trigger := range version.Schema.Triggers {
			if trigger.SkelName == config.TriggerSkelName {
				vpre.Check(len(trigger.Arguments) == 0, "scheduled task trigger %s must have no arguments", config.TriggerSkelName)
				return
			}
		}
	}
	vpre.Panicf("scheduled task trigger %s not found on task %s", config.TriggerSkelName, config.TaskSkelName)
}

func (s *Scheduler) publishSchedule(config _ScheduleConfig) {
	if !s.hasActiveRunner(config) {
		return
	}

	trace := meta.InitialTrace()
	s.publisher.PublishTask(taskspec.NATSMessage{
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
	})
}

func (s *Scheduler) hasActiveRunner(config _ScheduleConfig) bool {
	for _, status := range s.RegistryRepo.ListAppStatuses() {
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
