package main

import (
    "os"
    "log"
    "time"
    "regexp"
    "encoding/json"
    "net/http"
    "net/url"
    "io/ioutil"
    "github.com/rcrowley/go-metrics"
)

type VersionInfo struct {
    LastUpdateCheckTime time.Time
    StatusCode int
    VersionNumber *string
    Data []byte
}

func GetVersionInfo(url string) VersionInfo {
    resp, err := http.Get(url)

    badVersionInfo := func(status int, data []byte) VersionInfo {
        return VersionInfo {
            time.Now(),
            status,
            nil,
            data,
        }
    }

    if err != nil {
        log.Print(err)
        return badVersionInfo(500, nil)
    }

    defer resp.Body.Close()

    data, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        log.Print(err)
        return badVersionInfo(500, nil)
    }

    log.Printf("Got %d from %s", resp.StatusCode, url)

    if resp.StatusCode != 200 {
        return badVersionInfo(resp.StatusCode, data);
    }

    var decoded map[string]interface{}

    if err := json.Unmarshal(data, &decoded); err != nil {
        log.Print("Couldn't decode ", url, " : ", err)
        return badVersionInfo(resp.StatusCode, data);
    }

    ver := decoded["version"].(string)

    return VersionInfo {
        time.Now(),
        resp.StatusCode,
        &ver,
        data,
    }
}

func main() {
    var address string
    var urlregexp_str string

    if len(os.Getenv("SQUIRREL_ADDRESS")) > 0 {
        address = os.Getenv("SQUIRREL_ADDRESS")
    } else {
        address = ":3000"
    }

    if len(os.Getenv("SQUIRREL_REGEXP")) > 0 {
        urlregexp_str = os.Getenv("SQUIRREL_REGEXP")
    } else {
        urlregexp_str = ".*"
    }

    urlregexp, err := regexp.Compile(urlregexp_str);
    if err != nil {
        panic(err);
    }

    registry := metrics.NewRegistry()
    metrics.RegisterRuntimeMemStats(registry)
    go metrics.CaptureRuntimeMemStats(registry, 5 * time.Minute)

    forbiddenRequests := metrics.NewRegisteredCounter("forbiddenRequests", registry)
    errorRequests := metrics.NewRegisteredCounter("errorRequests", registry)
    upToDateResponses := metrics.NewRegisteredCounter("upToDateResponses", registry)
    outOfDateResponses := metrics.NewRegisteredCounter("outOfDateResponses", registry)

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
            forbiddenRequests.Inc(1)
            w.WriteHeader(http.StatusForbidden)
            return
        }

        versionInfo := GetVersionInfo(dest[0])

        if versionInfo.VersionNumber == nil {
            errorRequests.Inc(1)
            w.WriteHeader(versionInfo.StatusCode)
            if versionInfo.Data != nil {
                w.Write(versionInfo.Data)
            }
            return
        }

        if len(queryVersions) == 1 && *versionInfo.VersionNumber == queryVersions[0] {
            upToDateResponses.Inc(1)
            w.WriteHeader(http.StatusNoContent)
            return
        }

        outOfDateResponses.Inc(1)
        w.Write(versionInfo.Data)
    })

    http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        metrics.WriteJSONOnce(registry, w)
    })

    log.Printf("listening on '%s', accepting urls matching '%s'", address, urlregexp_str)

    log.Fatal(http.ListenAndServe(address, nil))

}
