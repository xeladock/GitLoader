package state_var

import "sync/atomic"

var (
	ManRun                 atomic.Bool
	IsActive               atomic.Bool
	IsManualDownloading    atomic.Bool // идёт ручная загрузка
	IsScheduledDownloading atomic.Bool // идёт загрузка по расписанию
	//OverridePath           bool
	//OverridePath = false
	// или один общий:
	// IsDownloading atomic.Bool // идёт любая загрузка
)
var OverridePath = false

// var IsDownloading atomic.Bool
func init() {
	ManRun.Store(false)
	IsActive.Store(false) // ← выполнится автоматически при первом импорте пакета
}

//var IsDownloading2 atomic.Bool = true
