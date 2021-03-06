package main

import (
  "bufio"
  "bytes"
  "flag"
  "fmt"
  "io"
  "io/ioutil"
  "math/rand"
  "net/http"
  "os"
  "regexp"
  "runtime"
  "sort"
  "strings"
  "time"
)

type Options struct {
  t      int64
  n      uint64
  c      uint64
  repeat uint64
  urls   []string
  random bool
  die    bool
  v      bool
}

var verbose = false
var URL_REGEX = regexp.MustCompile("http://[^:]+")

func Info(format string, a ...interface{}) {
  fmt.Printf(format+"\n", a...)
}

func Fatal(format string, a ...interface{}) {
  Info(format, a...)
  os.Exit(1)
}

func Verbose(format string, a ...interface{}) {
  if !verbose {
    return
  }
  fmt.Printf(format+"\n", a...)
}

type Result struct {
  done   bool
  ok     bool
  err    error
  status int
  t      int64
  size   int
}

func smack(client *http.Client, url string, die bool) (bool, int64, int, int, error) {
  beginT := time.Now().UnixNano()
  resp, err := client.Get(url)
  if err != nil {
    totalT := time.Now().UnixNano() - beginT
    Verbose("ERROR: could not hit " + url + " - " + err.Error())
    if die {
      panic("ERROR: could not hit " + url + " - " + err.Error())
    }
    return false, totalT, -1, 0, err
  }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    totalT := time.Now().UnixNano() - beginT
    Verbose("ERROR: could not hit " + url + " - " + err.Error())
    if die {
      panic("ERROR: could not hit " + url + " - " + err.Error())
    }
    return false, totalT, resp.StatusCode, 0, err
  }
  totalT := time.Now().UnixNano() - beginT
  return (resp.StatusCode == 200), totalT, resp.StatusCode, len(body), nil
}

func counter(opts *Options, request chan chan bool, die chan uint64) {
  i := uint64(0)
  j := uint64(0)
  if opts.t > 0 {
    beginT := time.Now().Unix()
    for {
      if j < opts.c {
        resp := <-request
        currT := time.Now().Unix()
        if currT-beginT > opts.t {
          resp <- true
          j++
        } else {
          resp <- false
          i++
        }
      } else {
        die <- i
        return
      }
    }
  }
  for {
    if j < opts.c {
      resp := <-request
      if i >= opts.n {
        resp <- true
        j++
      } else {
        resp <- false
        i++
        if opts.n >= 10 && i%(opts.n/10) == 0 {
          Verbose("%d requests complete", i)
        }
      }
    } else {
      if i >= opts.n {
        die <- opts.n
        return
      }
      die <- i
      return
    }
  }
}

type PrintableResult struct {
  count     uint64
  totalT    float64
  times     []float64
  totalSize uint64
  avgSize   float64
  min       float64
  p25       float64
  p50       float64
  avg       float64
  p75       float64
  p80       float64
  p85       float64
  p90       float64
  p95       float64
  p99       float64
  max       float64
}

func createPrintableResult(slice []*Result) *PrintableResult {
  if slice == nil || len(slice) == 0 {
    return nil
  }
  pr := PrintableResult{}
  pr.count = uint64(0)
  pr.totalT = float64(0)
  pr.times = []float64{}
  pr.totalSize = uint64(0)
  for _, result := range slice {
    pr.count++
    pr.totalT += float64(result.t)
    pr.times = append(pr.times, float64(result.t))
    pr.totalSize += uint64(result.size)
  }
  pr.avgSize = float64(pr.totalSize) / float64(pr.count)
  sort.Float64s(pr.times)
  pr.min = pr.times[0]
  pr.p25 = pr.times[int(float64(25.0/100.0)*float64(pr.count))]
  pr.p50 = pr.times[int(float64(50.0/100.0)*float64(pr.count))]
  pr.avg = pr.totalT / float64(pr.count)
  pr.p75 = pr.times[int(float64(75.0/100.0)*float64(pr.count))]
  pr.p80 = pr.times[int(float64(80.0/100.0)*float64(pr.count))]
  pr.p85 = pr.times[int(float64(85.0/100.0)*float64(pr.count))]
  pr.p90 = pr.times[int(float64(90.0/100.0)*float64(pr.count))]
  pr.p95 = pr.times[int(float64(95.0/100.0)*float64(pr.count))]
  pr.p99 = pr.times[int(float64(99.0/100.0)*float64(pr.count))]
  pr.max = pr.times[pr.count-1]
  return &pr
}

func printResults(opts *Options, results map[int][]*Result) {
  // compute results
  Info("")
  for key, slice := range results {
    if key < 0 {
      Info("ERRORS")
      pr := createPrintableResult(slice)
      Info("  count         : %d", pr.count)
      Info("  average time  : %f ms", pr.avg/1000000.0)
      Info("  combined time : %f ms", pr.totalT/1000000.0)
      counts := make(map[string]uint64)
      for _, result := range slice {
        errName := bytes.NewBufferString(result.err.Error())
        if !opts.v {
          errName = bytes.NewBuffer(URL_REGEX.ReplaceAll(errName.Bytes(), []byte{}))
        }
        errNameStr := errName.String()
        count, ok := counts[errNameStr]
        if !ok {
          counts[errNameStr] = 1
        } else {
          counts[errNameStr] = count + 1
        }
      }
      for err, count := range counts {
        Info("  count of \"%s\" : %d", err, count)
      }
      Info("")
      Info("  min : %f ms", pr.min/1000000.0)
      Info("  p25 : %f ms", pr.p25/1000000.0)
      Info("  p50 : %f ms", pr.p50/1000000.0)
      Info("  p75 : %f ms", pr.p75/1000000.0)
      Info("  p80 : %f ms", pr.p80/1000000.0)
      Info("  p85 : %f ms", pr.p85/1000000.0)
      Info("  p90 : %f ms", pr.p90/1000000.0)
      Info("  p95 : %f ms", pr.p95/1000000.0)
      Info("  p99 : %f ms", pr.p99/1000000.0)
      Info("  max : %f ms", pr.max/1000000.0)
    } else {
      Info("HTTP Status %d", key)
      pr := createPrintableResult(slice)
      Info("  count         : %d", pr.count)
      Info("  average time  : %f ms", pr.avg/1000000.0)
      Info("  combined time : %f ms", pr.totalT/1000000.0)
      Info("  average size  : %f bytes", pr.avgSize)
      Info("  combined size : %d bytes", pr.totalSize)
      Info("")
      Info("  min : %f ms", pr.min/1000000.0)
      Info("  p25 : %f ms", pr.p25/1000000.0)
      Info("  p50 : %f ms", pr.p50/1000000.0)
      Info("  p75 : %f ms", pr.p75/1000000.0)
      Info("  p80 : %f ms", pr.p80/1000000.0)
      Info("  p85 : %f ms", pr.p85/1000000.0)
      Info("  p90 : %f ms", pr.p90/1000000.0)
      Info("  p95 : %f ms", pr.p95/1000000.0)
      Info("  p99 : %f ms", pr.p99/1000000.0)
      Info("  max : %f ms", pr.max/1000000.0)
    }
  }
}

func results(opts *Options, result chan *Result, done chan bool) {
  results := make(map[int][]*Result)
  for {
    res := <-result
    if res.done {
      printResults(opts, results)
      done <- true
      return
    }
    slice, ok := results[res.status]
    if !ok {
      results[res.status] = []*Result{res}
    } else {
      results[res.status] = append(slice, res)
    }
  }
}

func urls(opts *Options, out chan string) {
  if opts.random {
    rand.Seed(time.Now().UnixNano())
    for {
      out <- opts.urls[rand.Intn(len(opts.urls))]
    }
  } else {
    i := 0
    for {
      out <- opts.urls[i]
      i++
      if i >= len(opts.urls) {
        i = 0
      }
    }
  }
}

func user(opts *Options, num uint64, counter chan chan bool, res chan *Result, urls chan string) {
  client := &http.Client{
	  Transport: &http.Transport{},
  }
  ch := make(chan bool)
  for {
    counter <- ch
    die := <-ch
    if die {
      close(ch)
      return
    }
    ok, t, status, size, err := smack(client, <-urls, opts.die)
    res <- &Result{done: false, ok: ok, t: t, status: status, size: size, err: err}
  }
}

func readLines(path string) (lines []string, err error) {
  var (
    file   *os.File
    part   []byte
    prefix bool
  )
  if file, err = os.Open(path); err != nil {
    return
  }
  defer file.Close()
  reader := bufio.NewReader(file)
  buffer := bytes.NewBuffer(make([]byte, 0))
  for {
    if part, prefix, err = reader.ReadLine(); err != nil {
      break
    }
    buffer.Write(part)
    if !prefix {
      lines = append(lines, buffer.String())
      buffer.Reset()
    }
  }
  if err == io.EOF {
    err = nil
  }
  return
}

func usage() {
  fmt.Printf("%s [options] (url|file)+\n", os.Args[0])
  fmt.Print("(url|file)+: a space separated list of urls and/or files containing a newline separated list of urls\n")
  fmt.Print("options:\n")
  fmt.Print(" -c: (uint var) the number of users smacking (concurrency)\n")
  fmt.Print(" -t: (int var) the number of seconds to continue smacking.\n")
  fmt.Print(" -n: (uint var) the total number of smacks (or number of repetitions if -t is used)\n")
  fmt.Print(" -r: (bool flag) if specified, will pick a random url from those specified for each request\n")
  fmt.Print(" -v: (bool flag) if specified, will output more information. useful for debugging but hinders performance\n")
  fmt.Print(" -p: (bool flag) if specified, smack will panic if an error (not bad http status) occurs while trying to request a url\n")
  fmt.Printf("e.g. \"%s -n 10000 -c 100 -r /tmp/urls.txt\" will smack the urls in /tmp/urls.txt randomly for a total number of 10000 requests with 100 users smacking at a time.\n", os.Args[0])
}

func main() {
  // make sure we use all our CPUs
  runtime.GOMAXPROCS(runtime.NumCPU())

  // parse flags
  opts := Options{}
  flag.Uint64Var(&opts.n, "n", uint64(1), "number of requests or repetitions")
  flag.Int64Var(&opts.t, "t", int64(0), "number of seconds to smack")
  flag.Uint64Var(&opts.c, "c", uint64(1), "concurrency")
  flag.BoolVar(&opts.random, "r", false, "randomize the urls")
  flag.BoolVar(&verbose, "v", false, "verbose (hinders performance numbers)")
  flag.BoolVar(&opts.die, "p", false, "panic if error")
  flag.Usage = usage
  flag.Parse()
  if flag.NArg() == 0 {
    flag.Usage()
    Fatal("ERROR: no urls or files specified")
  }
  opts.v = verbose
  Info("Preparing Smacker...")
  urlsOrFiles := flag.Args()
  opts.urls = []string{}
  for _, urlOrFile := range urlsOrFiles {
    if strings.HasPrefix(urlOrFile, "http://") || strings.HasPrefix(urlOrFile, "https://") {
      opts.urls = append(opts.urls, urlOrFile)
    } else {
      urls, err := readLines(urlOrFile)
      if err != nil {
        Fatal("ERROR: invalid file: %s", err.Error())
      }
      for _, url := range urls {
        opts.urls = append(opts.urls, url)
      }
    }
  }
  if opts.t > 0 {
    opts.repeat = opts.n
  } else {
    opts.repeat = 1
  }

  urlsCh := make(chan string)
  go urls(&opts, urlsCh)

  for i := uint64(0); i < opts.repeat; i++ {
    counterCh := make(chan chan bool)
    counterDoneCh := make(chan uint64)
    resultCh := make(chan *Result)
    resultDoneCh := make(chan bool)
    go results(&opts, resultCh, resultDoneCh)
    go counter(&opts, counterCh, counterDoneCh)
    Info("\n-------- Run %d --------", i+1)
    Info("Smacking Things...")
    beginT := time.Now().UnixNano()
    for i := uint64(0); i < opts.c; i++ {
      go user(&opts, i, counterCh, resultCh, urlsCh)
    }
    numRequests := <-counterDoneCh
    totalT := time.Now().UnixNano() - beginT
    Info("Formatting Results...")
    close(counterCh)
    close(counterDoneCh)
    resultCh <- &Result{done: true}
    <-resultDoneCh
    Info("")
    Info("requests took %f seconds", (float64(totalT) / 1000000000.0))
    Info("%f requests/sec", float64(numRequests)/(float64(totalT)/1000000000.0))
    close(resultCh)
    close(resultDoneCh)
  }
}
