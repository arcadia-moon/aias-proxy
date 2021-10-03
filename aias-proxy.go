package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

type flushWriter struct {
	w *bufio.ReadWriter
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	fw.w.Flush()
	return
}

func handler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "CONNECT" {
		fmt.Println("CONNECT request")

		remoteConn, err := net.Dial("tcp", req.Host)
		if err != nil {
			panic(err)
		}

		rw.Header()["Date"] = nil
		rw.Header()["Content-Type"] = nil
		rw.Header()["Transfer-Encoding"] = nil
		rw.WriteHeader(http.StatusOK)

		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			http.Error(rw, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, buffrw, err := hijacker.Hijack()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		readFromClient := func() {
			io.Copy(remoteConn, buffrw)
		}

		go readFromClient()
		io.Copy(flushWriter{buffrw}, remoteConn)

	} else {
		fmt.Println("Regular HTTP request")

		newreq, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		for name, values := range req.Header {
			// TODO: Don't copy hop-by-hop headers
			for _, value := range values {
				newreq.Header.Add(name, value)
			}
		}

		resp, err := http.DefaultClient.Do(newreq)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(resp.StatusCode)
		io.Copy(rw, resp.Body)
	}
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(handler)))
}
