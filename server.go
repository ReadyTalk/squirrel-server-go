package main

import (
    "log"
    "flag"
    "time"
    "regexp"
    "encoding/json"
    "net/http"
    "net/url"
    "io/ioutil"
)

type VersionInfo struct {
    LastUpdateCheckTime time.Time
    StatusCode int
    VersionNumber string
    Data []byte
}

func GetVersionInfo(url string) VersionInfo {
    resp, err := http.Get(url)

    badVersionInfo := func(status int, data []byte) VersionInfo {
        return VersionInfo {
            time.Now(),
            status,
            "<unknown>",
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

    return VersionInfo {
        time.Now(),
        resp.StatusCode,
        decoded["version"].(string),
        data,
    }
}

func main() {
    address := flag.String("address", ":3000", "Address to listen on")
    urlregexp_str := flag.String("regexp", ".*", "Regular expressoin for valid urls to proxy")
    flag.Parse()

    urlregexp, err := regexp.Compile(*urlregexp_str);
    if err != nil {
        panic(err);
    }

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

        versionInfo := GetVersionInfo(dest[0])

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
