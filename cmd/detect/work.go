package detect

import (
	"sort"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/cmd/detect/tasks"
	"github.com/evcc-io/evcc/util"
	"github.com/fatih/structs"
	"github.com/jeremywohl/flatten"
)

func workers(log *util.Logger, num int, tasks <-chan string, hits chan<- []tasks.Result) *sync.WaitGroup {
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

func workunit(log *util.Logger, ips <-chan string, hits chan<- []tasks.Result) {
	for ip := range ips {
		res := taskList.Test(log, "", tasks.ResultDetails{IP: ip})
		hits <- res
	}
}

func Work(log *util.Logger, num int, hosts []string) []tasks.Result {
	ip := make(chan string)
	hits := make(chan []tasks.Result)
	done := make(chan struct{})

	// log.INFO.Println(
	// 	"\n" +
	// 		strings.Join(
	// 			lo.Map(taskList.tasks, func(t tasks.Task) string {
	// 				return fmt.Sprintf("task: %s\ttype: %s\tdepends: %s\n", t.ID, t.Type, t.Depends)
	// 			}).([]string),
	// 			"",
	// 		),
	// )

	wg := workers(log, num, ip, hits)

	var res []tasks.Result
	go func() {
		for hits := range hits {
			res = append(res, hits...)
		}
		done <- struct{}{}
	}()

	for _, host := range hosts {
		ip <- host
	}

	close(ip)
	wg.Wait()

	close(hits)
	<-done

	return postProcess(res)
}

func postProcess(res []tasks.Result) []tasks.Result {
	for idx, hit := range res {
		// if sma, ok := hit.Details.(SmaResult); ok {
		// 	hit.Host = sma.Addr
		// }

		hit.Attributes = make(map[string]interface{})
		flat, _ := flatten.Flatten(structs.Map(hit), "", flatten.DotStyle)
		for k, v := range flat {
			hit.Attributes[strings.ToLower(k)] = v
		}

		res[idx] = hit
	}

	// sort by host
	sort.Slice(res, func(i, j int) bool {
		if res[i].ResultDetails.IP == res[j].ResultDetails.IP {
			return res[i].Type < res[j].Type
		}
		return res[i].ResultDetails.IP < res[j].ResultDetails.IP
	})

	return res
}
