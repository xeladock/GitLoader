package config

type AppConfig struct {
	GitLabLogin    string `json:"gitlab_login"`
	GitLabPass     string `json:"gitlab_pass_hash"`
	NetboxToken    string `json:"netbox_token_hash"`
	Mode           string `json:"mode"`
	Parser         string `json:"parser"`
	TargetMode     string `json:"target_mode"`
	SaveMode       string `json:"save_mode"`
	ScheduleDays   int    `json:"schedule_days"`
	ScheduleTime   string `json:"schedule_time"`
	SavedPlace     string `json:"saved_place"`
	LastRun        string `json:"last_run,omitempty"`
	SchedulerState string `json:"scheduler_state"`
}
