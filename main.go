package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Color uint8

const (
	Blue Color = iota
	Green
)

func file(color Color) string {
	switch color {
	case Blue:
		return "docker-compose.blu.yml"
	case Green:
		return "docker-compose.grn.yml"
	default:
		panic("unknown color")
	}
}

func port(color Color) string {
	switch color {
	case Blue:
		return "8080"
	case Green:
		return "8081"
	default:
		panic("fuck")
	}
}

var grn *exec.Cmd
var blu *exec.Cmd

func up(ctx context.Context, color Color) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", file(color), "up", "--remove-orphans", "--pull", "always")
	switch color {
	case Blue:
		blu = cmd
	case Green:
		grn = cmd
	}

	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Println(err)
	}
}

func down(color Color) {
	var cmd *exec.Cmd
	switch color {
	case Blue:
		cmd = blu
	case Green:
		cmd = grn
	}

	if err := cmd.Cancel(); err != nil {
		log.Println(err)
	}
}

var mtx sync.Mutex

func main() {
	ctx := context.Background()

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	cert := os.Getenv("HTTPS_CERT_PATH")
	key := os.Getenv("HTTPS_KEY_PATH")

	color := Blue
	go up(ctx, Blue)

	http.HandleFunc("POST /sentinel/deploy", func(w http.ResponseWriter, r *http.Request) {
		if !hmac.Equal([]byte(r.Header.Get("X-Sentinel-Auth")), []byte(os.Getenv("SECRET_KEY"))) {
			log.Println(r.Header.Get("X-Sentinel-Auth"))
			log.Println(os.Getenv("SECRET_KEY"))
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		log.Println("auth worked")

		mtx.Lock()
		defer mtx.Unlock()

		if color == Blue {
			go up(ctx, Green)
		} else {
			go up(ctx, Blue)
		}
		w.WriteHeader(http.StatusAccepted)

		ticker := time.NewTicker(time.Second).C
		timeout := time.NewTimer(300 * time.Second).C
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker:
			case <-timeout:
				return
			}
			p := port(1 - color)
			_, err := http.Get(fmt.Sprintf("http://localhost:%v/", p))
			if err != nil {
				log.Println("err connecting to new deployment:", err)
				continue
			}
			color = 1 - color
			break
		}

		if color == Blue {
			down(Green)
		} else {
			down(Blue)
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "deployed: %v", color)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var port string
		switch color {
		case Blue:
			port = "8080"
		case Green:
			port = "8081"
		default:
			panic("unknown color")
		}
		var buf bytes.Buffer
		buf.WriteString("http://localhost:")
		buf.WriteString(port)
		buf.WriteString(r.URL.Path)
		if len(r.URL.RawQuery) > 0 {
			buf.WriteString("?")
			buf.WriteString(r.URL.RawQuery)
		}
		req, err := http.NewRequest(r.Method, buf.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Copy headers
		req.Header = r.Header

		// Forward the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		for k, v := range resp.Header {
			w.Header()[k] = v
		}

		// Copy response status code
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		io.Copy(w, resp.Body)
	})

	if os.Getenv("ENVIRONMENT") == "PROD" {
		log.Println("listening on :443")
		err = http.ListenAndServeTLS(":443", cert, key, nil)
		if err != nil {
			log.Fatalln("http.ListenAndServeTLS:", err)
		}
	} else {
		log.Println("listening on :3000")
		err = http.ListenAndServe(":3000", nil)
		if err != nil {
			log.Fatalln("http.ListenAndServe:", err)
		}
	}
}
