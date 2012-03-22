package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type response struct {
	SearchResponse struct {
		Image struct {
			Results []struct {
				MediaUrl    string
				ContentType string
			}
		}
	}
}

func main() {
	if len(os.Args) != 4 {
		println("usage: oppai [appid] [outdir] [keyword]")
		os.Exit(1)
	}
	appid, outdir, keyword := os.Args[1], os.Args[2], os.Args[3]

	total := 0
	offset := 0
	outdir, _ = filepath.Abs(outdir)
	param := url.Values{
		"AppId":       {appid},
		"Version":     {"2.2"},
		"Market":      {"ja-JP"},
		"Sources":     {"Image"},
		"Image.Count": {strconv.Itoa(50)},
		"Adult":       {"off"},
		"Query":       {keyword},
	}
	quit := make(chan bool)

	md5hash := md5.New()
	to_filename := func(s, t string) string {
		md5hash.Reset()
		md5hash.Write([]byte(s))
		token := strings.SplitN(t, "/", 2)
		if strings.Index(token[1], "jpeg") != -1 {
			token[1] = "jpg"
		}
		return fmt.Sprintf("%X.%s", md5hash.Sum(nil), token[1])
	}

	for {
		param["Image.Offset"] = []string{strconv.Itoa(offset)}
		res, err := http.Get("http://api.bing.net/json.aspx?" +
			param.Encode())
		count := 0
		if err == nil {
			var result *response
			err = json.NewDecoder(res.Body).Decode(&result)
			res.Body.Close()
			if err != nil {
				println(err.Error())
				break
			}
			if count = len(result.SearchResponse.Image.Results); count ==
				0 {
				total = -1
				break
			}
			for _, r := range result.SearchResponse.Image.Results {
				go func(url, ct string) {
					filename := filepath.Join(outdir, to_filename(url, ct))
					if f, derr := os.Create(filename); derr == nil {
						defer f.Close()
						dres, derr := http.Get(url)
						if derr == nil && dres.ContentLength > 0 &&
							strings.Index(dres.Header.Get("Content-Type"), "image/") == 0 {
							_, derr = io.CopyN(f, dres.Body, dres.ContentLength)
							if derr != nil {
								println(derr.Error())
							} else {
								println(filename)
							}
						}
					}
					quit <- false
				}(r.MediaUrl, r.ContentType)
			}
		} else {
			total = -1
			break
		}
		offset += count
		total += count
	}

	for total > 0 {
		<-quit
		total--
		println(total)
	}
}
