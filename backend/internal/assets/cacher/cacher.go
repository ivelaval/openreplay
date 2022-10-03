package cacher

import (
	"context"
	"crypto/tls"
	"fmt"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"openreplay/backend/pkg/monitoring"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	config "openreplay/backend/internal/config/assets"
	"openreplay/backend/pkg/storage"
	"openreplay/backend/pkg/url/assets"
)

const MAX_CACHE_DEPTH = 5

type cacher struct {
	timeoutMap       *timeoutMap      // Concurrency implemented
	s3               *storage.S3      // AWS Docs: "These clients are safe to use concurrently."
	httpClient       *http.Client     // Docs: "Clients are safe for concurrent use by multiple goroutines."
	rewriter         *assets.Rewriter // Read only
	Errors           chan error
	sizeLimit        int
	downloadedAssets syncfloat64.Counter
	requestHeaders   map[string]string
	workers          *WorkerPool
}

func NewCacher(cfg *config.Config, metrics *monitoring.Metrics) *cacher {
	rewriter := assets.NewRewriter(cfg.AssetsOrigin)
	if metrics == nil {
		log.Fatalf("metrics are empty")
	}
	downloadedAssets, err := metrics.RegisterCounter("assets_downloaded")
	if err != nil {
		log.Printf("can't create downloaded_assets metric: %s", err)
	}
	c := &cacher{
		timeoutMap: newTimeoutMap(),
		s3:         storage.NewS3(cfg.AWSRegion, cfg.S3BucketAssets),
		httpClient: &http.Client{
			Timeout: time.Duration(6) * time.Second,
			Transport: &http.Transport{
				Proxy:           http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		rewriter:         rewriter,
		Errors:           make(chan error),
		sizeLimit:        cfg.AssetsSizeLimit,
		downloadedAssets: downloadedAssets,
		requestHeaders:   cfg.AssetsRequestHeaders,
	}
	c.workers = NewPool(32, c.CacheFile)
	return c
}

func (c *cacher) CacheFile(task *Task) {
	c.cacheURL(task.requestURL, task.sessionID, task.depth, task.urlContext, task.isJS)
}

func (c *cacher) cacheURL(requestURL string, sessionID uint64, depth byte, urlContext string, isJS bool) {
	var cachePath string
	if isJS {
		cachePath = assets.GetCachePathForJS(requestURL)
	} else {
		cachePath = assets.GetCachePathForAssets(sessionID, requestURL)
	}
	if c.timeoutMap.contains(cachePath) {
		return
	}
	c.timeoutMap.add(cachePath)
	crTime := c.s3.GetCreationTime(cachePath)
	if crTime != nil && crTime.After(time.Now().Add(-MAX_STORAGE_TIME)) { // recently uploaded
		return
	}

	req, _ := http.NewRequest("GET", requestURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; rv:31.0) Gecko/20100101 Firefox/31.0")
	for k, v := range c.requestHeaders {
		req.Header.Set(k, v)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		c.Errors <- errors.Wrap(err, urlContext)
		return
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		// TODO: retry
		c.Errors <- errors.Wrap(fmt.Errorf("Status code is %v, ", res.StatusCode), urlContext)
		return
	}
	data, err := ioutil.ReadAll(io.LimitReader(res.Body, int64(c.sizeLimit+1)))
	if err != nil {
		c.Errors <- errors.Wrap(err, urlContext)
		return
	}
	if len(data) > c.sizeLimit {
		c.Errors <- errors.Wrap(errors.New("Maximum size exceeded"), urlContext)
		return
	}

	contentType := res.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(res.Request.URL.Path))
	}
	isCSS := strings.HasPrefix(contentType, "text/css")

	strData := string(data)
	if isCSS {
		strData = c.rewriter.RewriteCSS(sessionID, requestURL, strData) // TODO: one method for rewrite and return list
	}

	// TODO: implement in streams
	err = c.s3.Upload(strings.NewReader(strData), cachePath, contentType, false)
	if err != nil {
		c.Errors <- errors.Wrap(err, urlContext)
		return
	}
	c.downloadedAssets.Add(context.Background(), 1)

	if isCSS {
		if depth > 0 {
			for _, extractedURL := range assets.ExtractURLsFromCSS(string(data)) {
				if fullURL, cachable := assets.GetFullCachableURL(requestURL, extractedURL); cachable {
					go c.cacheURL(fullURL, sessionID, depth-1, urlContext+"\n  -> "+fullURL, false)
				}
			}
			if err != nil {
				c.Errors <- errors.Wrap(err, urlContext)
				return
			}
		} else {
			c.Errors <- errors.Wrap(errors.New("Maximum recursion cache depth exceeded"), urlContext)
			return
		}
	}
	return
}

func (c *cacher) CacheJSFile(sourceURL string) {
	c.workers.AddTask(&Task{
		requestURL: sourceURL,
		sessionID:  0,
		depth:      0,
		urlContext: sourceURL,
		isJS:       true,
	})
	//go c.cacheURL(sourceURL, 0, 0, sourceURL, true)
}

func (c *cacher) CacheURL(sessionID uint64, fullURL string) {
	c.workers.AddTask(&Task{
		requestURL: fullURL,
		sessionID:  sessionID,
		depth:      MAX_CACHE_DEPTH,
		urlContext: fullURL,
		isJS:       false,
	})
	//go c.cacheURL(fullURL, sessionID, MAX_CACHE_DEPTH, fullURL, false)
}

func (c *cacher) UpdateTimeouts() {
	c.timeoutMap.deleteOutdated()
}

func (c *cacher) Stop() {
	c.workers.Stop()
}

type Task struct {
	requestURL string
	sessionID  uint64
	depth      byte
	urlContext string
	isJS       bool
}

type WorkerPool struct {
	tasks chan *Task
	wg    sync.WaitGroup
	done  chan struct{}
	term  sync.Once
	size  int
	job   Job
}

type Job func(task *Task)

func NewPool(size int, job Job) *WorkerPool {
	newPool := &WorkerPool{
		tasks: make(chan *Task, 64),
		done:  make(chan struct{}),
		size:  size,
		job:   job,
	}
	newPool.init()
	return newPool
}

func (p *WorkerPool) init() {
	p.wg.Add(p.size)
	for i := 0; i < p.size; i++ {
		go p.worker()
	}
}

func (p *WorkerPool) worker() {
	for {
		select {
		case newTask := <-p.tasks:
			log.Printf("handle new task: %+v", newTask)
			p.job(newTask)
		case <-p.done:
			p.wg.Done()
			return
		}
	}
}

func (p *WorkerPool) AddTask(newTask *Task) {
	p.tasks <- newTask
}

func (p *WorkerPool) Stop() {
	log.Printf("stopping workers")
	p.term.Do(func() {
		close(p.done)
	})
	p.wg.Wait()
	log.Printf("all workers have been stopped")
}
