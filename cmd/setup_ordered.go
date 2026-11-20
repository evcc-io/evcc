package cmd

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

// orderedItem is a device whose instantiation may depend on other devices of the same class
type orderedItem struct {
	name string
	refs []string
	run  func() error
}

// runOrdered instantiates items concurrently, deferring each item until the items it references are done.
// Later waves still run after a failure so every device ends up registered and editable.
func runOrdered(items []orderedItem) error {
	var res error
	pending := items

	for len(pending) > 0 {
		isPending := func(name string) bool {
			return slices.ContainsFunc(pending, func(it orderedItem) bool { return it.name == name })
		}

		var ready, deferred []orderedItem
		for _, it := range pending {
			if slices.ContainsFunc(it.refs, isPending) {
				deferred = append(deferred, it)
			} else {
				ready = append(ready, it)
			}
		}

		if len(ready) == 0 {
			names := lo.Map(deferred, func(it orderedItem, _ int) string { return it.name })
			return joinErrors(res, fmt.Errorf("circular device references: %v", names))
		}

		var eg errgroup.Group
		for _, it := range ready {
			eg.Go(it.run)
		}
		if err := eg.Wait(); err != nil && res == nil {
			res = err
		}

		pending = deferred
	}

	return res
}
