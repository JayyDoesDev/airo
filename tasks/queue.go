package taskqueue

import (
	"log"
	"sync/atomic"
	"time"
)

type Task struct {
	Name        string
	GuildID     string
	UserID      string
	Action      string
	Reason      string
	Role        string
	DMContent   string
	ResponseMsg string

	EmbedTitle        string
	EmbedDescription  string
	EmbedThumbnailUrl string
	EmbedImageUrl     string
	UseEmbed          bool

	Execute func() error
}

type Queue struct {
	tasks  chan Task
	quit   chan struct{}
	active int32
}

func NewQueue(bufferSize int) *Queue {
	return &Queue{
		tasks: make(chan Task, bufferSize),
		quit:  make(chan struct{}),
	}
}

func (q *Queue) Start(workerName string) {
	go func() {
		for {
			select {
			case task := <-q.tasks:
				log.Printf("[%s] Executing task: %s for user %s", workerName, task.Action, task.UserID)
				atomic.AddInt32(&q.active, 1)
				err := task.Execute()
				atomic.AddInt32(&q.active, -1)
				if err != nil {
					log.Printf("[%s] Task failed: %v", workerName, err)
				}
				time.Sleep(500 * time.Millisecond)
			case <-q.quit:
				log.Printf("[%s] Shutting down task queue", workerName)
				return
			}
		}
	}()
}

func (q *Queue) Add(task Task) {
	q.tasks <- task
}

func (q *Queue) Len() int {
	return len(q.tasks)
}

func (q *Queue) Active() int {
	return int(atomic.LoadInt32(&q.active))
}

func (q *Queue) Stop() {
	close(q.quit)
}
