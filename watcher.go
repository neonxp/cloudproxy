package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type watcher struct {
	sync.Map
	cl  client.APIClient
	upd chan interface{}
}

func newWatcher() (*watcher, error) {
	cl, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return &watcher{
		cl: cl,
	}, nil
}

func (w *watcher) watch(ctx context.Context) error {
	for {
		containers, err := w.cl.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			return err
		}
		toDelete := map[string]struct{}{}
		w.Range(func(key, value interface{}) bool {
			toDelete[key.(string)] = struct{}{}
			return true
		})
		for _, container := range containers {
			newHost := host{}
			for k, v := range container.Labels {
				switch k {
				case "cp.host":
					newHost.Host = v
				case "cp.port":
					port, err := strconv.Atoi(v)
					if err != nil {
						return fmt.Errorf("can't parse port of container %s : %w", container.ID, err)
					}
					newHost.Port = port
				case "cp.tls":
					newHost.TLS = true
				}
			}
			if newHost.Host == "" || newHost.Port == 0 {
				continue
			}
			nc := container.NetworkSettings.Networks["bridge"]
			newHost.Addr = nc.IPAddress
			_, loaded := w.LoadOrStore(newHost.Host, newHost)
			if !loaded {
				log.Println("registered", newHost.Host, "->", newHost.Addr, newHost.Port)
				w.upd <- nil
			}
			delete(toDelete, newHost.Host)
		}
		for host := range toDelete {
			w.Delete(host)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(15 * time.Second):
			continue
		}
	}
}

func (w *watcher) update() <-chan interface{} {
	return w.upd
}
