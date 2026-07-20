package taskqueue

import "time"

var BotQueue = NewQueue(100)
var StartTime time.Time
