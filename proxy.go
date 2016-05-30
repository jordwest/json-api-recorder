package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
)

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type customTransport struct{}

func (t *customTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	response, err := transport.RoundTrip(request)
	//response, err := http.DefaultTransport.RoundTrip(request)

	if response.Header.Get("Content-Type") == "application/json" {
		body, err := httputil.DumpResponse(response, true)
		if err != nil {
			// copying the response body did not work
			return nil, err
		}
		filename := strings.Replace(path.Join("./", request.URL.Path)+".json", "/", "-", -1)
		fmt.Println("Writing", filename)

		ioutil.WriteFile(filename, body, os.ModePerm)
	}

	return response, err
}

// Provides a reverse proxy for intercepting all requests before forwarding them on, and recording each response
func interceptRequest(targetURL string, host string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		req.Host = host
		req.Header.Del("Accept-Encoding")
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		fmt.Printf("%s %s \n\t\t--> %s [Host: %s]\n",
			req.Method, req.URL.RequestURI(),
			target.Host, req.Host)
	}

	/*
		transport := &customTransport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: acceptInvalidCert,
			},
		}
	*/
	transport := &customTransport{}

	return &httputil.ReverseProxy{
		Director:  director,
		Transport: transport,
	}, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	targetURL := "https://192.0.78.23"
	target, err := url.Parse(targetURL)
	if err != nil {
		w.WriteHeader(500)
		fmt.Println(err)
		return
	}

	fmt.Printf("%s %s\n", r.Method, r.URL.RequestURI())

	c := http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	proxyReq, err := http.NewRequest(r.Method, r.URL.RequestURI(), r.Body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Println(err)
		return
	}
	proxyReq.URL.Scheme = target.Scheme
	proxyReq.URL.Host = target.Host
	proxyReq.Host = "public-api.wordpress.com"
	proxyReq.Header.Set("Cookie", r.Header.Get("Cookie"))
	proxyReq.Header.Set("Authorization", r.Header.Get("Authorization"))

	fmt.Printf("\t\t--> %s %s [Host: %s]\n",
		proxyReq.Method, proxyReq.URL.RequestURI(),
		proxyReq.Header.Get("Host"))
	resp, err := c.Do(proxyReq)
	if err != nil {
		w.WriteHeader(500)
		fmt.Println(err)
		return
	}
	fmt.Println(resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Println(err)
		return
	}

	if resp.Header.Get("Content-Type") == "application/json" && resp.StatusCode == 200 {
		filename := strings.Replace(path.Join("./", r.URL.Path)+".json", "/", "-", -1)
		fmt.Println("Writing", filename)

		ioutil.WriteFile(filename, body, os.ModePerm)
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Set-Cookie", resp.Header.Get("Set-Cookie"))
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
