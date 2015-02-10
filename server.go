package main

import (
    "io/ioutil"
    "log"
    "os"
    "flag"
    "strings"
    "net/http"
    "net/url"
    "path/filepath"
)

func main() {
    address := flag.String("address", ":3000", "Address to listen on")
    dir := flag.String("dir", "releases", "Directory to serve version data from")
    flag.Parse()

    filepath.Walk(*dir, func(path string, f os.FileInfo, err error) error {
        if err != nil {
            panic(err)
        }

        if filepath.Ext(path) == ".json" {
            servePath := filepath.ToSlash(strings.TrimPrefix(filepath.Dir(path), *dir))
            if !strings.HasPrefix(servePath, "/") {
                servePath = "/" + servePath
            }
            version := strings.TrimSuffix(filepath.Base(path), ".json")
            data, _ := ioutil.ReadFile(path)
            log.Printf("serving %s version %s", servePath, version)
            http.HandleFunc(servePath, func(w http.ResponseWriter, r *http.Request) {
                query, _ := url.ParseQuery(r.URL.RawQuery)
                queryVersions := query["version"]
                if len(queryVersions) > 0 && queryVersions[0] == version {
                    w.WriteHeader(204)
                } else {
                    w.Header().Add("Content-Type", "application/json")
                    w.Write(data)
                }
            })
        }

        return nil
    })

    log.Fatal(http.ListenAndServe(*address, nil))

}
