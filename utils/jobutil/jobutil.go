package jobutil

import (
	"fmt"

	pb "github.com/jackie8tao/ezjob/proto"
)

func JobKey(name string) string {
	return fmt.Sprintf("%s/%s", pb.JobKey, name)
}

func TriggerKey(name string) string {
	return fmt.Sprintf("%s/%s", pb.TriggerKey, name)
}
