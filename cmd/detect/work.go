package detect

import (
	"sync"

	"github.com/andig/evcc/util"
)

type Result struct {
	Task
	Host       string
	Details    interface{}
	Attributes map[string]interface{}
}

func workers(log *util.Logger, num int, tasks <-chan string, hits chan<- []Result) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			workunit(log, tasks, hits)
			wg.Done()
		}()
	}

	return &wg
}

func workunit(log *util.Logger, tasks <-chan string, hits chan<- []Result) {
	for ip := range tasks {
		res := taskList.Test(log, ip)
		hits <- res
	}
}

func Work(log *util.Logger, num int, hosts []string) []Result {
	tasks := make(chan string)
	hits := make(chan []Result)
	done := make(chan struct{})
	wg := workers(log, num, tasks, hits)

	var res []Result
	go func() {
		for hits := range hits {
			res = append(res, hits...)
		}
		done <- struct{}{}
	}()

	for _, host := range hosts {
		tasks <- host
	}

	close(tasks)
	wg.Wait()

	close(hits)
	<-done

	return res
}
