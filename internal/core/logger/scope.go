package logger

import (
	"log/slog"
	"sync/atomic"

	"go.yorun.ai/vine/util/vpre"
)

type Subsystem string

const (
	SubsystemRpcServer Subsystem = "rpc-server"
	SubsystemTask      Subsystem = "task"
	SubsystemEvent     Subsystem = "event"
)

func IsValidSubsystem(subsystem Subsystem) bool {
	return subsystem == SubsystemRpcServer ||
		subsystem == SubsystemTask ||
		subsystem == SubsystemEvent
}

type Scope struct {
	// AppName is the exact, case-sensitive ApplicationSpec name of the local owner.
	AppName string
	// Subsystem is empty for ordinary App logs or one of the closed framework subsystems.
	Subsystem Subsystem
}

type AppSubsystemLevel struct {
	// AppName is the exact, case-sensitive ApplicationSpec name.
	AppName string
	// Subsystem identifies the framework lifecycle scope.
	Subsystem Subsystem
	// Level is the threshold for this App plus subsystem selector.
	Level Level
}

type LevelOverrides struct {
	// Subsystems maps subsystem selectors to thresholds.
	Subsystems map[Subsystem]Level
	// Apps maps exact App names to thresholds.
	Apps map[string]Level
	// AppSubsystem contains exact App plus subsystem selectors.
	AppSubsystem []AppSubsystemLevel
}

type _AppSubsystemKey struct {
	appName   string
	subsystem Subsystem
}

type _LevelSnapshot struct {
	subsystems   map[Subsystem]Level
	apps         map[string]Level
	appSubsystem map[_AppSubsystemKey]Level
}

var levelSnapshot atomic.Pointer[_LevelSnapshot]

func init() {
	levelSnapshot.Store(new(_LevelSnapshot{
		subsystems:   map[Subsystem]Level{},
		apps:         map[string]Level{},
		appSubsystem: map[_AppSubsystemKey]Level{},
	}))
}

type _ScopeLeveler struct {
	scope Scope
}

func (l *_ScopeLeveler) Level() slog.Level {
	scope := l.scope
	snapshot := levelSnapshot.Load()
	if scope.AppName != "" && scope.Subsystem != "" {
		if level, ok := snapshot.appSubsystem[_AppSubsystemKey{appName: scope.AppName, subsystem: scope.Subsystem}]; ok {
			return level.ToSLogLevel()
		}
	}
	if scope.AppName != "" {
		if level, ok := snapshot.apps[scope.AppName]; ok {
			return level.ToSLogLevel()
		}
	}
	if scope.Subsystem != "" {
		if level, ok := snapshot.subsystems[scope.Subsystem]; ok {
			return level.ToSLogLevel()
		}
	}
	return globalLevel.Level()
}

func SetSubsystemLevel(subsystem Subsystem, level Level) {
	validateSubsystemLevel(subsystem, level)
	updateLevelSnapshot(func(next *_LevelSnapshot) {
		next.subsystems[subsystem] = level
	})
}

func ClearSubsystemLevel(subsystem Subsystem) {
	validateSubsystem(subsystem)
	updateLevelSnapshot(func(next *_LevelSnapshot) {
		delete(next.subsystems, subsystem)
	})
}

func SetAppLevel(appName string, level Level) {
	validateAppLevel(appName, level)
	updateLevelSnapshot(func(next *_LevelSnapshot) {
		next.apps[appName] = level
	})
}

func ClearAppLevel(appName string) {
	validateAppName(appName)
	updateLevelSnapshot(func(next *_LevelSnapshot) {
		delete(next.apps, appName)
	})
}

func SetAppSubsystemLevel(appName string, subsystem Subsystem, level Level) {
	validateAppLevel(appName, level)
	validateSubsystem(subsystem)
	updateLevelSnapshot(func(next *_LevelSnapshot) {
		next.appSubsystem[_AppSubsystemKey{appName: appName, subsystem: subsystem}] = level
	})
}

func ClearAppSubsystemLevel(appName string, subsystem Subsystem) {
	validateAppName(appName)
	validateSubsystem(subsystem)
	updateLevelSnapshot(func(next *_LevelSnapshot) {
		delete(next.appSubsystem, _AppSubsystemKey{appName: appName, subsystem: subsystem})
	})
}

func ReplaceLevelOverrides(overrides LevelOverrides) {
	next := new(_LevelSnapshot{
		subsystems:   make(map[Subsystem]Level, len(overrides.Subsystems)),
		apps:         make(map[string]Level, len(overrides.Apps)),
		appSubsystem: make(map[_AppSubsystemKey]Level, len(overrides.AppSubsystem)),
	})
	for subsystem, level := range overrides.Subsystems {
		validateSubsystemLevel(subsystem, level)
		next.subsystems[subsystem] = level
	}
	for appName, level := range overrides.Apps {
		validateAppLevel(appName, level)
		next.apps[appName] = level
	}
	for _, entry := range overrides.AppSubsystem {
		validateAppLevel(entry.AppName, entry.Level)
		validateSubsystem(entry.Subsystem)
		next.appSubsystem[_AppSubsystemKey{appName: entry.AppName, subsystem: entry.Subsystem}] = entry.Level
	}
	levelSnapshot.Store(next)
}

func updateLevelSnapshot(update func(*_LevelSnapshot)) {
	for {
		current := levelSnapshot.Load()
		next := cloneLevelSnapshot(current)
		update(next)
		if levelSnapshot.CompareAndSwap(current, next) {
			return
		}
	}
}

func cloneLevelSnapshot(source *_LevelSnapshot) *_LevelSnapshot {
	next := new(_LevelSnapshot{
		subsystems:   make(map[Subsystem]Level, len(source.subsystems)),
		apps:         make(map[string]Level, len(source.apps)),
		appSubsystem: make(map[_AppSubsystemKey]Level, len(source.appSubsystem)),
	})
	for subsystem, level := range source.subsystems {
		next.subsystems[subsystem] = level
	}
	for appName, level := range source.apps {
		next.apps[appName] = level
	}
	for key, level := range source.appSubsystem {
		next.appSubsystem[key] = level
	}
	return next
}

func validateSubsystemLevel(subsystem Subsystem, level Level) {
	validateSubsystem(subsystem)
	validateLevel(level)
}

func validateAppLevel(appName string, level Level) {
	validateAppName(appName)
	validateLevel(level)
}

func validateSubsystem(subsystem Subsystem) {
	vpre.Check(IsValidSubsystem(subsystem), "%+v is not a valid LogSubsystem", subsystem)
}

func validateAppName(appName string) {
	vpre.Check(appName != "", "logger app name cannot be empty")
}

func validateLevel(level Level) {
	vpre.Check(IsValidLevel(level), "%+v is not a valid LogLevel", level)
}
