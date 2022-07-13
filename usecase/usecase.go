package usecase

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Srishti24Jain/Load-Balancer/domain"

	constant "github.com/Srishti24Jain/Load-Balancer/const"
)

var serverPool LoadBalancer

// Backend holds the data about a server
type Backend struct {
	URL          *url.URL
	mux          sync.RWMutex
	Alive        bool
	addr         string
	ReverseProxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	backends []*Backend
	current  uint64

	Port            string
	roundRobinCount int
	servers         []domain.Server
}

func (b *Backend) Serve(rw http.ResponseWriter, req *http.Request) {
	b.ReverseProxy.ServeHTTP(rw, req)
}

func RegisterUrl(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)

	var backend domain.RegisterUrls
	err := json.Unmarshal(reqBody, &backend)
	if err != nil {
		return
	}

	file, _ := json.MarshalIndent(backend, "", " ")

	_ = ioutil.WriteFile("config.json", file, 0644)
	err = json.NewEncoder(w).Encode(backend)
	if err != nil {
		return
	}
}

func NewSimpleServer(addr string) *Backend {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	serverPool.AddBackend(&Backend{
		URL:          serverUrl,
		Alive:        true,
		ReverseProxy: proxyUrl(serverUrl),
	})

	return &Backend{
		URL:          serverUrl,
		addr:         addr,
		Alive:        true,
		ReverseProxy: proxyUrl(serverUrl),
	}
}

func NewLoadBalancer(port string, servers []domain.Server) *LoadBalancer {
	return &LoadBalancer{
		Port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func (l *LoadBalancer) ServeProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := l.GetNextPeer()

	targetServer.Serve(rw, req)
}

// GetNextPeer returns next active peer to take a connection
func (l *LoadBalancer) GetNextPeer() *Backend {

	// loop entire backends to find out an Alive backend
	next := l.NextIndex()

	// start from next and move a full cycle
	length := len(serverPool.backends) + next

	for i := next; i < length; i++ {
		idx := i % len(serverPool.backends)
		if serverPool.backends[idx].IsAlive() { // if we have an alive backend, use it and store if it's not the original one
			if i != next {
				atomic.StoreUint64(&serverPool.current, uint64(idx))
			}
			return serverPool.backends[idx]
		}
	}
	return nil
}

// NextIndex atomically increase the counter and return an index
func (l *LoadBalancer) NextIndex() int {
	return int(atomic.AddUint64(&serverPool.current, uint64(1)) % uint64(len(serverPool.backends)))
}

func proxyUrl(serverUrl *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(serverUrl)
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
		request.URL.Host = serverUrl.Host
		request.URL.Scheme = serverUrl.Scheme

		log.Printf("Error[%s] %s\n", serverUrl.Host, e.Error())
		retries := GetRetryFromContext(request)
		if retries < 3 {
			select {
			case <-time.After(10 * time.Millisecond):
				ctx := context.WithValue(request.Context(), constant.Retry, retries+1)
				proxy.ServeHTTP(writer, request.WithContext(ctx))
			}
			return
		}

		// after 3 retries, mark this backend as down
		serverPool.MarkBackendStatus(serverUrl, false)

		// if the same request routing for few attempts with different backends, increase the count
		attempts := GetAttemptsFromContext(request)
		log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
		ctx := context.WithValue(request.Context(), constant.Attempts, attempts+1)
		loadBalance(writer, request.WithContext(ctx))
	}
	return proxy
}

// loadBalance load balances the incoming request
func loadBalance(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// AddBackend to the server pool
func (l *LoadBalancer) AddBackend(backend *Backend) []*Backend {
	l.backends = append(l.backends, backend)
	return l.backends
}

// HealthCheck runs a routine for check status of the backends every 15 sec
func HealthCheck() {
	t := time.NewTicker(time.Second * 15)
	for {
		select {
		case <-t.C:
			log.Println("Starting health check...")
			serverPool.Check()
			log.Println("Health check completed")
		}
	}
}

// Check pings the backends and update the status
func (l *LoadBalancer) Check() {
	for _, b := range l.backends {
		status := "up"
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}

// isBackendAlive checks whether a backend is Alive by establishing a TCP connection
func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host+":"+u.Scheme, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)
	return true
}

// SetAlive for this backend
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) Address() string {
	return b.addr
}
func (b *Backend) IsAlive() bool { return true }

// MarkBackendStatus changes a status of a backend
func (l *LoadBalancer) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for _, b := range l.backends {
		if b.URL.String() == backendUrl.String() {
			b.SetAlive(alive)
			break
		}
	}
}

// GetAttemptsFromContext returns the attempts for request
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(constant.Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetRetryFromContext - GetAttemptsFromContext returns the attempts for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(constant.Retry).(int); ok {
		return retry
	}
	return 0
}

func handleErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}
