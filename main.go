package main

import (
	"fmt"
	"github.com/qingsong-he/ce"
	"github.com/qingsong-he/swf"
	"go.uber.org/zap"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// default http recover no http status code and body content, so don't use it
func withRecover(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inCome := time.Now()

		defer func() {
			if err := recover(); err != nil {
				if !ce.IsFromMe(err) {
					ce.Error("panic", zap.Any("errByPanic", err))
				}
				if _, ok := err.(error); ok {
					http.Error(w, err.(error).Error(), http.StatusInternalServerError)
				} else {
					http.Error(w, fmt.Sprintf("%#v", err), http.StatusInternalServerError)
				}
				return
			}

			ce.Info("", zap.String("addr", r.RemoteAddr), zap.String("m", r.Method), zap.String("h", r.Host), zap.String("url", r.RequestURI), zap.Duration("d", time.Since(inCome)))
		}()

		next.ServeHTTP(w, r)
		return
	}
}

func main() {
	var wg sync.WaitGroup

	srvExitAlarm := make(chan struct{})
	httpSrv := swf.NewSwf(":3000")

	// root router
	httpSrv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("it works"))
	}, withRecover)

	// test router
	httpSrv.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if time.Now().Unix()%2 == 0 {
			w.Write([]byte("/hello"))
		} else {
			panic(1)
			//ce.CheckError(io.EOF)
		}
	}, withRecover)

	// pprof router
	httpSrv.HandleFunc("/debug/pprof/", pprof.Index, withRecover)
	httpSrv.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline, withRecover)
	httpSrv.HandleFunc("/debug/pprof/profile", pprof.Profile, withRecover)
	httpSrv.HandleFunc("/debug/pprof/symbol", pprof.Symbol, withRecover)
	httpSrv.HandleFunc("/debug/pprof/trace", pprof.Trace, withRecover)

	// static file router
	httpSrv.HandleFunc("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/"))).ServeHTTP, withRecover)

	wg.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				if !ce.IsFromMe(err) {
					ce.Error("panic", zap.Any("errByPanic", err))
				}
			}
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					if !ce.IsFromMe(err) {
						ce.Error("panic", zap.Any("errByPanic", err))
					}
				}
				wg.Done()
			}()
			<-srvExitAlarm
			err := httpSrv.Stop()
			ce.CheckError(err)
		}()

		err := httpSrv.Run()
		ce.CheckError(err)
	}()

	var mainByExitAlarm chan os.Signal
	mainByExitAlarm = make(chan os.Signal, 1)
	signal.Notify(mainByExitAlarm, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP)

forLableByNotify:
	for {
		s := <-mainByExitAlarm
		ce.Info(s.String())
		switch s {
		case syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM:
			break forLableByNotify

		case syscall.SIGHUP:
		default:
			break forLableByNotify
		}
	}

	close(srvExitAlarm) // close http service
	wg.Wait()

	// flush and close log
	ce.Sync()
}
