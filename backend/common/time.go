package common

import "time"

func GetTimestamp() int64 {
	return time.Now().Unix()
}
