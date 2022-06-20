package proto

const EventKey = "events:ezjob"

const (
	RoleFollower = "follower"
	RoleLeader   = "leader"
	RoleDeader   = "deader"
)

const (
	ElectionKey = "/ezjob/election"
	JobKey      = "/ezjob/tasks"
	TriggerKey  = "/ezjob/triggers"
)

const (
	DispatchTypeHttp  = "http"
	DispatchTypeKafka = "kafka"
)

const (
	JobStatusFail    = "fail"
	JobStatusSuccess = "success"
	TaskStatusDoing  = "doing"
)

const (
	TaskModeSingle = "single"
	TaskModeMulti  = "multi"
)
