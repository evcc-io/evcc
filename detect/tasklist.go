package detect

import (
	"fmt"
	"sync"

	"github.com/andig/evcc/util"
)

type Task struct {
	ID, Type string
	Depends  string
	Config   map[string]interface{}
}

type TaskList struct {
	tasks    []Task
	handlers []TaskHandler
	once     sync.Once
}

func (l *TaskList) Add(task Task) {
	l.tasks = append(l.tasks, task)
}

func (l *TaskList) Count() int {
	return len(l.tasks)
}

func (l *TaskList) delete(i int) {
	if len(l.tasks) == 1 {
		l.tasks = nil
	}

	res := l.tasks[:i]
	if i < len(l.tasks)-1 {
		res = append(res, l.tasks[i+1:]...)
	}

	l.tasks = res
}

func (l *TaskList) sort() {
	var res []Task

	for len(l.tasks) > 0 {
		last := len(l.tasks)

	NEXT:
		for i, task := range l.tasks {
			if task.Depends == "" {
				res = append(res, task)
				l.delete(i)
				break NEXT
			}

			for _, sortedTask := range res {
				if task.Depends == sortedTask.ID {
					res = append(res, task)
					l.delete(i)
					break NEXT
				}
			}
		}

		if last == len(l.tasks) {
			panic("tasks with unmatched dependencies: " + fmt.Sprintf("%v", l))
		}
	}

	l.tasks = res
}

func (l *TaskList) createHandlers() {
	for _, task := range l.tasks {
		factory, err := registry.Get(task.Type)
		if err != nil {
			panic("invalid task type " + task.Type)
		}

		handler, err := factory(task.Config)
		if err != nil {
			panic("invalid config: " + err.Error())
		}

		l.handlers = append(l.handlers, handler)
	}
}

func (l *TaskList) Test(log *util.Logger, ip string) (res []Result) {
	l.once.Do(func() {
		l.sort()
		l.createHandlers()
	})

	failed := make(map[string]bool)

HANDLERS:
	for id, handler := range l.handlers {
		task := l.tasks[id]

		for failure := range failed {
			if failure == task.Depends {
				continue HANDLERS
			}
		}

		results := handler.Test(log, ip)
		if len(results) > 0 {
			log.INFO.Printf("ip: %s task: %s ok", ip, task.ID)

			for _, detail := range results {
				res = append(res, Result{
					Task:    task,
					Host:    ip,
					Details: detail,
				})
			}
		} else {
			// log.INFO.Printf("ip: %s task: %s nok", ip, task.ID)
			failed[task.ID] = true
		}
	}

	return res
}
