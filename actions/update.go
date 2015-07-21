package actions

import (
	"math/rand"

	"github.com/CenturyLinkLabs/watchtower/container"
)

var (
	letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func allContainersFilter(container.Container) bool { return true }

func Update() error {
	client := container.NewClient()
	containers, err := client.ListContainers(allContainersFilter)
	if err != nil {
		return err
	}

	for i := range containers {
		if err := client.RefreshImage(&containers[i]); err != nil {
			return err
		}
	}

	containers, err = container.SortByDependencies(containers)
	if err != nil {
		return err
	}

	checkDependencies(containers)

	// Stop stale containers in reverse order
	for i := len(containers) - 1; i >= 0; i-- {
		container := containers[i]

		if container.IsWatchtower() {
			break
		}

		if container.Stale {
			if err := client.Stop(container, 10); err != nil {
				return err
			}
		}
	}

	// Restart stale containers in sorted order
	for _, container := range containers {
		if container.Stale {
			// Since we can't shutdown a watchtower container immediately, we need to
			// start the new one while the old one is still running. This prevents us
			// from re-using the same container name so we first rename the current
			// instance so that the new one can adopt the old name.
			if container.IsWatchtower() {
				if err := client.Rename(container, randName()); err != nil {
					return err
				}
			}

			if err := client.Start(container); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkDependencies(containers []container.Container) {

	for i, parent := range containers {
		if parent.Stale {
			continue
		}

	LinkLoop:
		for _, linkName := range parent.Links() {
			for _, child := range containers {
				if child.Name() == linkName && child.Stale {
					containers[i].Stale = true
					break LinkLoop
				}
			}
		}
	}
}

// Generates a random, 32-character, Docker-compatible container name.
func randName() string {
	b := make([]rune, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}