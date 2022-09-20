package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/go-chi/chi"
	pinpoint "github.com/pinpoint-apm/pinpoint-go-agent"
	pchi "github.com/pinpoint-apm/pinpoint-go-agent/plugin/chi"
	phttp "github.com/pinpoint-apm/pinpoint-go-agent/plugin/http"
)

func hello(w http.ResponseWriter, r *http.Request) {
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	time.Sleep(time.Duration(random.Intn(5000)+1) * time.Millisecond)
	io.WriteString(w, "hello world")
}

func shutdown(w http.ResponseWriter, r *http.Request) {
	pinpoint.GetAgent().Shutdown()
	io.WriteString(w, "shutdown")
}

func outgoing(w http.ResponseWriter, r *http.Request) {
	ctx := pinpoint.NewContext(context.Background(), pinpoint.FromContext(r.Context()))
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:9000/async_wrapper", nil)

	resp, err := phttp.DoClient(http.DefaultClient.Do, req)
	if nil != err {
		io.WriteString(w, err.Error())
		return
	}

	defer resp.Body.Close()
	io.Copy(w, resp.Body)

	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	time.Sleep(time.Duration(random.Intn(2000)+1) * time.Millisecond)
}

func myMiddleware(s string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(s)
			h.ServeHTTP(w, r)
		})
	}
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	opts := []pinpoint.ConfigOption{
		pinpoint.WithAppName("GoChiTest"),
		pinpoint.WithAgentId("GoChiTestAgent"),
		//pinpoint.WithSamplingCounterRate(10),
		pinpoint.WithConfigFile(os.Getenv("HOME") + "/tmp/pinpoint-config.yaml"),
	}

	cfg, _ := pinpoint.NewConfig(opts...)
	agent, _ := pinpoint.NewAgent(cfg)
	//if err != nil {
	//	log.Fatalf("pinpoint agent start fail: %v", err)
	//}
	defer agent.Shutdown()

	r := chi.NewRouter()
	r.Use(myMiddleware("myMiddleware"))
	r.Use(pchi.Middleware())

	r.Get("/hello", hello)
	r.Get("/outgoing", outgoing)
	r.Get("/shutdown", shutdown)

	http.ListenAndServe(":8000", r)
}
