package crawler

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/Sirupsen/logrus"
)

type CrawlItem struct {
	Url     string
	Retries int
	Result  Result
}

type Result struct {
	// The final (possibly redirected) url.
	Url string

	Error    error
	CanRetry bool

	Urls []string
}

type Crawler struct {
	sync.Mutex

	ConcurrentRequests int
	MaxRetries         int

	AllowCrawlPatterns []*regexp.Regexp

	Logger *logrus.Logger

	Client *http.Client

	itemQueue   []*CrawlItem
	crawledUrls map[string]bool
	pendingUrls map[string]bool

	activeRequests int
	resultChannel  chan *CrawlItem
}

func New(concurrentRequests int, hosts []string, startUrls []string) *Crawler {
	c := &Crawler{
		ConcurrentRequests: concurrentRequests,
		AllowCrawlPatterns: make([]*regexp.Regexp, 0),

		Logger: logrus.New(),

		Client: &http.Client{},

		itemQueue:   make([]*CrawlItem, 0),
		crawledUrls: make(map[string]bool),
		pendingUrls: make(map[string]bool),

		resultChannel: make(chan *CrawlItem),
	}

	for _, host := range hosts {
		re := regexp.MustCompile("https?\\://" + host + "*")
		c.AllowCrawlPatterns = append(c.AllowCrawlPatterns, re)
	}

	for _, url := range startUrls {
		c.itemQueue = append(c.itemQueue, &CrawlItem{
			Url: url,
		})
	}

	return c
}

func (c *Crawler) canCrawl(url string) bool {
	for _, re := range c.AllowCrawlPatterns {
		if re.MatchString(url) {
			return true
		}
	}

	return false
}

func (c *Crawler) AddUrls(urls []string) {
	for _, url := range urls {
		c.AddUrl(url)
	}
}

func (c *Crawler) AddUrl(url string) {
	_, wasCrawled := c.crawledUrls[url]
	_, isPending := c.pendingUrls[url]

	if c.canCrawl(url) && !wasCrawled && !isPending {
		c.itemQueue = append(c.itemQueue, &CrawlItem{
			Url: url,
		})
	}
}

func (c *Crawler) Run() {
	for {
		select {
		case item := <-c.resultChannel:
			c.activeRequests -= 1
			delete(c.pendingUrls, item.Url)

			if item.Result.Error != nil {
				// If url was redirected, set redirected url to crawled.
				if item.Result.Url != "" && item.Result.Url != item.Url {
					c.crawledUrls[item.Result.Url] = true
				}

				// Crawl failed.
				if item.Retries >= c.MaxRetries || !item.Result.CanRetry {
					c.Logger.Errorf("Crawl error at %v: %v", item.Url, item.Result.Error)
				} else {
					item.Retries += 1
					c.itemQueue = append(c.itemQueue, item)
				}
			} else {
				c.Logger.Debugf("Crawl succeeded: %v", item.Url)
				c.crawledUrls[item.Url] = true
				if item.Result.Url != item.Url {
					c.crawledUrls[item.Result.Url] = true
				}
				c.AddUrls(item.Result.Urls)
				// Set urls to nil to garbage collect the memory.
				item.Result.Urls = nil
			}
		default:
		}

		for c.activeRequests < c.ConcurrentRequests && len(c.itemQueue) > 0 {
			item := c.itemQueue[0]
			c.itemQueue = c.itemQueue[1:]

			_, wasCrawled := c.crawledUrls[item.Url]
			_, isPending := c.pendingUrls[item.Url]

			if !wasCrawled && !isPending {
				c.pendingUrls[item.Url] = true
				c.crawlItem(item)
			}
		}

		if len(c.pendingUrls) == 0 && len(c.itemQueue) == 0 && c.activeRequests == 0 {
			break
		}
	}

	c.Logger.Info("Crawler finished")
}

func normalizeUrl(response *http.Response, rawUrl string) string {
	if rawUrl == "" {
		return ""
	}

	if !strings.HasPrefix(rawUrl, "http") {
		url, err := url.Parse(rawUrl)
		if err != nil {
			return ""
		}

		requestUrl := response.Request.URL

		url.Scheme = requestUrl.Scheme
		url.Host = requestUrl.Host
		url.Fragment = ""

		return url.String()
	}

	return rawUrl
}

func (c *Crawler) crawlItem(item *CrawlItem) {
	c.activeRequests += 1
	go func() {
		resp, err := c.Client.Get(item.Url)

		if err != nil {
			item.Result.Error = err
			item.Result.CanRetry = true

			c.resultChannel <- item
			return
		}

		item.Result.Url = resp.Request.URL.String()

		// Build goquery document.
		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			item.Result.Error = err
			c.resultChannel <- item
			return
		}

		// Extract urls.
		urls := make([]string, 0)
		doc.Find("a").Each(func(i int, a *goquery.Selection) {
			href, ok := a.Attr("href")
			if ok {
				url := normalizeUrl(resp, href)
				if url != "" {
					urls = append(urls, url)
				}
			}
		})
		item.Result.Urls = urls

		c.resultChannel <- item
	}()
}
