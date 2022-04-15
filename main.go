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
			t = []terminator{}
			err = fmt.Errorf("failed to setup containers: %w", err)
		}
	}()

	r, err := setupRedis(ctx)
	if err != nil {
		return
	}
	t = append(t, r.Container)

	wd, err := os.Getwd()
	if err != nil {
		return
	}
	dir := path.Join(wd, "pipeline")

	b, err := setupBenthos(ctx, path.Join(dir, "dynamic_ingest"), map[string]string{"REDIS_URI": r.URI})
	if err != nil {
		return
	}

	_, err = setupBenthos(ctx, path.Join(dir, "wiki_scraper"), map[string]string{"REDIS_URI": r.URI, "DYNAMIC_INGEST_URL": b.URI })

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
