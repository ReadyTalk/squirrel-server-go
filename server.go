package main

import (
    // "io/ioutil"
    "log"
    // "os"
    "flag"
    "time"
    "regexp"
    "encoding/json"
    "sync"
    "net/http"
    "net/url"
    "io/ioutil"
    // "path/filepath"
)

type VersionInfo struct {
    LastUpdateCheckTime time.Time
    StatusCode int
    VersionNumber string
    Data []byte
}

type VersionCache struct {
    cache map [string] *VersionInfo
}

func NewVersionCache() *VersionCache {
    return &VersionCache{make(map [string] *VersionInfo)}
}

func (vc *VersionCache) GetOrLookup(url string) VersionInfo {
    cached := vc.cache[url]
    if cached == nil || time.Since(cached.LastUpdateCheckTime) > 30 * time.Second {
        resp, err := http.Get(url)

        if err != nil {
            log.Print(err)
            return VersionInfo {
                time.Now(),
                500,
                "<unknown>",
                nil,
            }
        }

        data, err := ioutil.ReadAll(resp.Body)

        if err != nil {
            log.Print(err)
            return VersionInfo {
                time.Now(),
                500,
                "<unknown>",
                nil,
            }
        }

        log.Printf("Got %d from %s", resp.StatusCode, url)

        if resp.StatusCode != 200 {
            return VersionInfo {
                time.Now(),
                resp.StatusCode,
                "<unknown>",
                data,
            }
        }

        var decoded map[string]interface{}

        if err := json.Unmarshal(data, &decoded); err != nil {
            log.Print("Couldn't decode ", url, " : ", err)
            return VersionInfo {
                time.Now(),
                resp.StatusCode,
                "<unknown>",
                data,
            }
        }

        versionInfo := VersionInfo {
            time.Now(),
            resp.StatusCode,
            decoded["version"].(string),
            data,
        }

        cached = &versionInfo
        vc.cache[url] = cached
    }

    return *cached
}

func main() {
    address := flag.String("address", ":3000", "Address to listen on")
    urlregexp_str := flag.String("regexp", ".*", "Regular expressoin for valid urls to proxy")
    flag.Parse()

    urlregexp, err := regexp.Compile(*urlregexp_str);
    if err != nil {
        panic(err);
    }

    cache := NewVersionCache()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        query, _ := url.ParseQuery(r.URL.RawQuery)
        queryVersions := query["version"]
        dest := query["url"]

        if len(dest) != 1 || len(queryVersions) > 1 {
            w.WriteHeader(http.StatusBadRequest)
            return
        }

        valid := urlregexp.MatchString(dest[0])

        if !valid {
            w.WriteHeader(http.StatusForbidden)
            return
        }

        versionInfo := cache.GetOrLookup(dest[0])

        if versionInfo.StatusCode != 200 {
            w.WriteHeader(versionInfo.StatusCode)
            if versionInfo.Data != nil {
                w.Write(versionInfo.Data)
            }
            return
        }

        if len(queryVersions) == 1 && versionInfo.VersionNumber == queryVersions[0] {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        w.Write(versionInfo.Data)
    })

    log.Fatal(http.ListenAndServe(*address, nil))

}
