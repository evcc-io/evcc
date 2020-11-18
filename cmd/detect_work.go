package cmd

import (
	"fmt"
	"sync"

	"github.com/andig/evcc/cmd/detect"
)

func workers(num int, tasks <-chan string, hits chan<- []detect.Result) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			workunit(tasks, hits)
			wg.Done()
		}()
	}

	return &wg
}

func workunit(tasks <-chan string, hits chan<- []detect.Result) {
	for ip := range tasks {
		res := taskList.Test(log, ip)
		fmt.Println(res)
		hits <- res
	}
}

func work(num int, hosts []string) []detect.Result {
	tasks := make(chan string)
	hits := make(chan []detect.Result)
	wg := workers(num, tasks, hits)

	var res []detect.Result
	go func() {
		for hits := range hits {
			res = append(res, hits...)
			fmt.Println(res)
		}
	}()

	for _, host := range hosts {
		tasks <- host
	}

	close(tasks)
	wg.Wait()

	fmt.Println(res)

	close(hits)
	return res
}
