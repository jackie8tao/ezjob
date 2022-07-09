package proto

import (
	"fmt"
)

func GenJobKey(name string) string {
	return fmt.Sprintf("%s/%s", JobKey, name)
}

func GenTriggerKey(name string) string {
	return fmt.Sprintf("%s/%s", TriggerKey, name)
}
