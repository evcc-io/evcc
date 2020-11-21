package cmd

import (
	"fmt"
	"net"
)

type TaskList []Task

func (l TaskList) delete(i int) TaskList {
	if len(l) == 1 {
		return []Task{}
	}

	res := l[:i]
	if i < len(l)-1 {
		res = append(res, l[i+1:]...)
	}

	return res
}

func (l TaskList) Sorted() (res TaskList) {
	for len(l) > 0 {
		last := len(l)

	NEXT:
		for i, task := range l {
			if task.Depends == "" {
				res = append(res, task)
				l = l.delete(i)
				break NEXT
			}

			for _, sortedTask := range res {
				if task.Depends == sortedTask.ID {
					res = append(res, task)
					l = l.delete(i)
					break NEXT
				}
			}
		}

		if last == len(l) {
			panic("tasks with unmatched dependencies: " + fmt.Sprintf("%v", l))
		}
	}

	return res
}

func (l TaskList) Test(ip net.IP) {
	for _, task := range l {
		// fmt.Println(task)

		factory, err := registry.Get(task.Type)
		if err != nil {
			panic("invalid task type " + task.Type)
		}

		handler, err := factory(task.Config)
		if err != nil {
			panic("invalid config: " + err.Error())
		}

		ok := handler.Test(ip)
		if ok {
			log.INFO.Printf("ip: %s task: %s ok", ip, task.ID)
		}

		if !ok {
			break
		}
	}
}
