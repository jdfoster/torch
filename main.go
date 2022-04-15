package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"
)

type terminator interface {
	Terminate(context.Context) error
}

func setup(ctx context.Context) (stop func(context.Context), err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*4)
	defer cancel()

	var t []terminator
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

	wd, err := os.Getwd()
	if err != nil {
		return
	}
	dir := path.Join(wd, "pipeline")

	r, err := setupRedis(ctx)
	if err != nil {
		return
	}
	t = append(t, r.Container)

	b, err := setupBenthos(ctx, "dynamic_ingest", path.Join(dir, "dynamic_ingest"), map[string]string{"REDIS_URI": r.URI})
	if err != nil {
		return
	}
	t = append(t, b.Container)

	w, err := setupBenthos(ctx, "wiki_scraper", path.Join(dir, "wiki_scraper"), map[string]string{"REDIS_URI": r.URI, "DYNAMIC_INGEST_URL": b.URI})
	if err != nil {
		return
	}
	t = append(t, w.Container)

	c, err := setupBenthos(ctx, "cache_last_event", path.Join(dir, "cache_last_event"), map[string]string{"REDIS_URI": r.URI})
	if err != nil {
		return
	}
	t = append(t, c.Container)

	return
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cancel, err := setup(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	<-ctx.Done()
	fmt.Println("shutting down")
	cancel(context.Background())
}
