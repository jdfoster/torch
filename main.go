package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"
)

func setup(ctx context.Context, pipelinePath string) (stop func(context.Context), err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*4)
	defer cancel()

	var t []container
	stop = func(ctx context.Context) {
		for _, c := range t {
			c.Terminate(ctx)
		}
	}

	defer func() {
		// terminate any created containers on error
		if err != nil {
			stop(ctx)
			err = fmt.Errorf("failed to setup containers: %w", err)
		}
	}()

	r, err := newRedisContainer(ctx)
	if err != nil {
		return
	}
	t = append(t, *r)

	ep := map[string]string{"REDIS_URI": r.URI}

	bc := []struct {
		name, confPath, provides string
		requires                 []string
	}{
		{
			name:     "dynamic_ingest",
			confPath: path.Join(pipelinePath, "dynamic_ingest"),
			provides: "DYNAMIC_INGEST_URI",
			requires: []string{"REDIS_URI"},
		},
		{
			name:     "wiki_scraper",
			confPath: path.Join(pipelinePath, "wiki_scraper"),
			provides: "WIKI_SCRAPER_URI",
			requires: []string{"REDIS_URI", "DYNAMIC_INGEST_URI"},
		},
		{
			name:     "cache_last_event",
			confPath: path.Join(pipelinePath, "cache_last_event"),
			provides: "CACHE_LAST_EVENT_URI",
			requires: []string{"REDIS_URI"},
		},
	}

	var c *container
	for _, b := range bc {
		env := make(map[string]string)
		for _, r := range b.requires {
			e, ok := ep[r]
			if !ok {
				err = fmt.Errorf("missing endpoint: %q", r)
				return
			}
			env[r] = e
		}

		req := benthosRequest{
			name:     b.name,
			confPath: b.confPath,
			env:      env,
		}

		c, err = newBenthosContainer(ctx, req)
		if err != nil {
			return
		}

		ep[b.provides] = c.URI
		t = append(t, *c)
	}

	return
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	wd, err := os.Getwd()
	if err != nil {
		return
	}
	dir := path.Join(wd, "pipeline")

	cancel, err := setup(ctx, dir)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	<-ctx.Done()
	fmt.Println("shutting down")
	cancel(context.Background())
}
