package app

import (
	"reflect"
	"time"

	"github.com/robfig/cron/v3"
	"go.yorun.ai/vine/util/vpre"
)

type RunnerTypeAdder func(reflect.Type, ...RunnerOption)

type RunnerOption interface {
	applyRunner(*_RunnerOptions)
}

type _RunnerOptionFunc func(*_RunnerOptions)

func (f _RunnerOptionFunc) applyRunner(options *_RunnerOptions) {
	f(options)
}

type _RunnerOptions struct {
	Timeout        time.Duration
	Concurrency    int
	NoRetry        bool
	CronSchedulers []RunnerCronScheduler
}

type _RunnerTypeEntry struct {
	kind    reflect.Type
	options _RunnerOptions
}

type RunnerCronScheduler struct {
	TriggerSkelName string
	CronExpr        string
}

const (
	defaultRunnerTimeout     = 30 * time.Second
	defaultRunnerConcurrency = 10
	defaultRunnerNoRetry     = false
)

func WithRunnerTimeout(timeout time.Duration) RunnerOption {
	vpre.Check(timeout > 0, "runner timeout must be greater than 0")
	return _RunnerOptionFunc(func(options *_RunnerOptions) {
		options.Timeout = timeout
	})
}

func WithRunnerConcurrency(concurrency int) RunnerOption {
	vpre.Check(concurrency > 0, "runner concurrency must be greater than 0")
	return _RunnerOptionFunc(func(options *_RunnerOptions) {
		options.Concurrency = concurrency
	})
}

func WithRunnerNoRetry() RunnerOption {
	return _RunnerOptionFunc(func(options *_RunnerOptions) {
		options.NoRetry = true
	})
}

func WithRunnerCronScheduler(triggerSkelName string, cronExpr string) RunnerOption {
	vpre.CheckNotEmpty(triggerSkelName, "runner cron scheduler trigger skel name is empty")
	vpre.CheckNotEmpty(cronExpr, "runner cron scheduler cron expr is empty")
	_, err := cron.ParseStandard(cronExpr)
	vpre.CheckNilError(err, "runner cron scheduler cron expr is invalid")
	return _RunnerOptionFunc(func(options *_RunnerOptions) {
		options.CronSchedulers = append(options.CronSchedulers, RunnerCronScheduler{
			TriggerSkelName: triggerSkelName,
			CronExpr:        cronExpr,
		})
	})
}

func newRunnerOptions(options []RunnerOption) _RunnerOptions {
	parsed := _RunnerOptions{
		Timeout:     defaultRunnerTimeout,
		Concurrency: defaultRunnerConcurrency,
		NoRetry:     defaultRunnerNoRetry,
	}
	for _, option := range options {
		option.applyRunner(&parsed)
	}
	return parsed
}
