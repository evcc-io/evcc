package detect

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/cmd/detect/tasks"
	"github.com/evcc-io/evcc/util"
)

type TaskList struct {
	tasks []tasks.Task
	once  sync.Once
}

func (l *TaskList) Add(task tasks.Task) {
	task.TaskHandler = l.handler(task)
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
	var res []tasks.Task

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

func (l *TaskList) handler(task tasks.Task) tasks.TaskHandler {
	factory, err := tasks.Get(task.Type)
	if err != nil {
		panic("invalid task type " + task.Type)
	}

	// fmt.Println(task)
	handler, err := factory(task.Config)
	if err != nil {
		panic("invalid config: " + err.Error())
	}

	return handler
}

func (l *TaskList) Test(log *util.Logger, id string, input tasks.ResultDetails) []tasks.Result {
	l.once.Do(l.sort)

	var all []tasks.Result
	var inputs []tasks.ResultDetails

	if id == "" {
		inputs = append(inputs, input)
	} else {
		var task tasks.Task
		for _, t := range l.tasks {
			if t.ID == id {
				task = t
				break
			}
		}

		inputs = task.Test(log, input)
		success := len(inputs) > 0
		log.DEBUG.Printf("task: %s %v -> %v", id, input, success)

		for _, detail := range inputs {
			all = append(all, tasks.Result{
				Task:          task,
				ResultDetails: detail,
			})
		}
	}

	// run dependent tasks
	for _, task := range l.tasks {
		if task.Depends == id {
			// fmt.Println("task:", task)
			for _, input := range inputs {
				// fmt.Println("input:", input)
				all = append(all, l.Test(log, task.ID, input)...)
			}
		}
	}

	return all
}
