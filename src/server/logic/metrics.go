package logic

import (
	"github.com/prometheus/client_golang/prometheus"
	"rss_parrot/shared"
	"time"
)

type IMetrics interface {
	StartWebRequestIn(label string) IRequestObserver
	StartApubRequestIn(label string) IRequestObserver
	StartApubRequestOut(label string) IRequestObserver
	FeedRequested(label string)
	FeedUpdated()
	NewPostSaved()
	ServiceStarted()
	TotalFollowers(count int)
	TootQueueLength(length int)
	CheckableFeedCount(count int)
}

type IRequestObserver interface {
	Finish()
}

type metrics struct {
	cfg                *shared.Config
	webRequestsIn      *prometheus.HistogramVec
	apubRequestsIn     *prometheus.HistogramVec
	apubRequestsOut    *prometheus.HistogramVec
	feedsRequested     *prometheus.CounterVec
	feedsUpdated       prometheus.Counter
	newPostsSaved      prometheus.Counter
	serviceStarted     prometheus.Counter
	totalFollowers     prometheus.Gauge
	tootQueueLength    prometheus.Gauge
	checkableFeedCount prometheus.Gauge
}

func NewMetrics(cfg *shared.Config) IMetrics {

	res := metrics{}
	res.cfg = cfg

	res.webRequestsIn = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "web_requests_in_duration",
		Help: "Duration in seconds of Web requests served.",
	}, []string{"label"})
	prometheus.Register(res.webRequestsIn)

	res.apubRequestsIn = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "apub_requests_in_duration",
		Help: "Duration in seconds of ActivityPub requests served.",
	}, []string{"label"})
	prometheus.Register(res.apubRequestsIn)

	res.apubRequestsOut = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "apub_requests_out_duration",
		Help: "Duration in seconds of ActivityPub requests made.",
	}, []string{"label"})
	prometheus.Register(res.apubRequestsOut)

	res.feedsRequested = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "feeds_requested",
		Help: "Number of feeds requested",
	}, []string{"label"})
	prometheus.Register(res.feedsRequested)

	res.feedsUpdated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "feeds_updated",
		Help: "Number of feeds updated",
	})
	prometheus.Register(res.feedsUpdated)

	res.newPostsSaved = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "new_posts_saved",
		Help: "Number of new posts saved",
	})
	prometheus.Register(res.newPostsSaved)

	res.serviceStarted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "service_started",
		Help: "Service has started up",
	})
	prometheus.Register(res.serviceStarted)

	res.totalFollowers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "total_follower_count",
		Help: "Total follower count of parrot accounts",
	})
	prometheus.Register(res.totalFollowers)

	res.tootQueueLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "toot_queue_length",
		Help: "Items in toot queue",
	})
	prometheus.Register(res.tootQueueLength)

	res.checkableFeedCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "checkable_feed_count",
		Help: "Number of feeds waiting to be checked",
	})
	prometheus.Register(res.checkableFeedCount)

	return &res
}

type requestObserver struct {
	label string
	start time.Time
	hgvec *prometheus.HistogramVec
}

func (ro *requestObserver) Finish() {
	now := time.Now()
	elapsed := float64(now.UnixMilli()-ro.start.UnixMilli()) / 1000.0
	ro.hgvec.WithLabelValues(ro.label).Observe(elapsed)
}

func (m *metrics) StartWebRequestIn(label string) IRequestObserver {
	return &requestObserver{label, time.Now(), m.webRequestsIn}
}

func (m *metrics) StartApubRequestIn(label string) IRequestObserver {
	return &requestObserver{label, time.Now(), m.apubRequestsIn}
}

func (m *metrics) StartApubRequestOut(label string) IRequestObserver {
	return &requestObserver{label, time.Now(), m.apubRequestsOut}
}

func (m *metrics) FeedRequested(label string) {
	m.feedsRequested.WithLabelValues(label).Add(1)
}

func (m *metrics) TootQueueLength(length int) {
	m.tootQueueLength.Set(float64(length))
}

func (m *metrics) FeedUpdated() {
	m.feedsUpdated.Add(1)
}

func (m *metrics) NewPostSaved() {
	m.newPostsSaved.Add(1)
}

func (m *metrics) ServiceStarted() {
	m.serviceStarted.Add(1)
}

func (m *metrics) TotalFollowers(count int) {
	m.totalFollowers.Set(float64(count))
}

func (m *metrics) CheckableFeedCount(count int) {
	m.checkableFeedCount.Set(float64(count))
}
