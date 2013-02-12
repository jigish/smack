package main

import (
  "net/http"
  "time"
  "fmt"
  "runtime"
  "flag"
  "strings"
  "os"
  "math/rand"
  "bufio"
  "bytes"
  "io"
  "sort"
  "io/ioutil"
)

type Options struct {
  n uint64
  c uint64
  urls []string
  random bool
  die bool
}

var verbose = false

func Info(format string, a ...interface{}) {
  fmt.Printf(format+"\n", a...)
}

func Fatal(format string, a ...interface{}) {
  Info(format, a...)
  os.Exit(1)
}

func Verbose(format string, a ...interface{}) {
  if (!verbose) {
    return
  }
  fmt.Printf(format+"\n", a...)
}

type Result struct {
  done bool
  ok bool
  err error
  status int
  t int64
  size int
}

func smack(url string, die bool) (bool, int64, int, int, error) {
  beginT := time.Now().UnixNano()
  resp, err := http.Get(url)
  if err != nil {
    totalT := time.Now().UnixNano() - beginT
    Verbose("ERROR: could not hit "+url+" - "+err.Error())
    if die {
      panic("ERROR: could not hit "+url+" - "+err.Error())
    }
    return false, totalT, -1, 0, err
  }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    totalT := time.Now().UnixNano() - beginT
    Verbose("ERROR: could not hit "+url+" - "+err.Error())
    if die {
      panic("ERROR: could not hit "+url+" - "+err.Error())
    }
    return false, totalT, resp.StatusCode, 0, err
  }
  totalT := time.Now().UnixNano() - beginT
  return (resp.StatusCode == 200), totalT, resp.StatusCode, len(body), nil
}

func counter(opts *Options, request chan chan bool, die chan uint64) {
  i := uint64(0)
  j := uint64(0)
  for {
    if j < opts.c {
      resp := <-request
      if (i >= opts.n) {
        resp <-true
        j++
      } else {
        resp <-false
        i++
        if i%(opts.n/10) == 0 {
          Verbose("%d requests complete", i)
        }
      }
    } else {
      if (i >= opts.n) {
        die <-opts.n
        return
      }
      die <-i
      return
    }
  }
}

type PrintableResult struct {
  count uint64
  totalT float64
  times []float64
  totalSize uint64
  avgSize float64
  min float64
  p25 float64
  p50 float64
  avg float64
  p75 float64
  p80 float64
  p85 float64
  p90 float64
  p95 float64
  p99 float64
  max float64
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
  pr.avgSize = float64(pr.totalSize)/float64(pr.count)
  sort.Float64s(pr.times)
  pr.min = pr.times[0]
  pr.p25 = pr.times[int(float64(25.0/100.0)*float64(pr.count))]
  pr.p50 = pr.times[int(float64(50.0/100.0)*float64(pr.count))]
  pr.avg = pr.totalT/float64(pr.count)
  pr.p75 = pr.times[int(float64(75.0/100.0)*float64(pr.count))]
  pr.p80 = pr.times[int(float64(80.0/100.0)*float64(pr.count))]
  pr.p85 = pr.times[int(float64(85.0/100.0)*float64(pr.count))]
  pr.p90 = pr.times[int(float64(90.0/100.0)*float64(pr.count))]
  pr.p95 = pr.times[int(float64(95.0/100.0)*float64(pr.count))]
  pr.p99 = pr.times[int(float64(99.0/100.0)*float64(pr.count))]
  pr.max = pr.times[pr.count-1]
  return &pr
}

func printResults(results map[int][]*Result) {
  // compute results
  Info("")
  for key, slice := range(results) {
    if (key < 0) {
      Info("ERRORS")
      pr := createPrintableResult(slice)
      Info("  count         : %d", pr.count)
      Info("  average time  : %f ms", pr.avg/1000000.0)
      Info("  combined time : %f ms", pr.totalT/1000000.0)
      counts := make(map[string]uint64)
      for _, result := range(slice) {
        count, ok := counts[result.err.Error()]
        if !ok {
          counts[result.err.Error()] = 1
        } else {
          counts[result.err.Error()] = count + 1
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
      printResults(results)
      done <-true
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
  if (opts.random) {
    rand.Seed(time.Now().UnixNano())
    for {
      out <- opts.urls[rand.Intn(len(opts.urls))]
    }
  } else {
    i := 0
    for {
      out <- opts.urls[i]
      i++
      if (i >= len(opts.urls)) {
        i = 0
      }
    }
  }
}

func user(opts *Options, num uint64, counter chan chan bool, res chan *Result, urls chan string) {
  ch := make(chan bool)
  for {
    counter <- ch
    die := <-ch
    if die {
      close(ch)
      return
    }
    ok, t, status, size, err := smack(<-urls, opts.die)
    res <-&Result{done: false, ok: ok, t: t, status: status, size: size, err: err}
  }
}

func readLines(path string) (lines []string, err error) {
    var (
        file *os.File
        part []byte
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

func main() {
  // make sure we use all our CPUs
  runtime.GOMAXPROCS(runtime.NumCPU())

  Info("Preparing Smacker...")
  // parse flags
  opts := Options{}
  flag.Uint64Var(&opts.n, "n", uint64(1), "number of requests")
  flag.Uint64Var(&opts.c, "c", uint64(1), "concurrency")
  flag.BoolVar(&opts.random, "r", false, "randomize the urls")
  flag.BoolVar(&verbose, "v", false, "verbose (hinders performance numbers)")
  flag.BoolVar(&opts.die, "p", false, "panic if error")
  // flag.Usage = usage
  flag.Parse()
  if flag.NArg() == 0 {
    flag.Usage()
    Fatal("ERROR: no urls or files specified")
  }
  urlsOrFiles := flag.Args()
  opts.urls = []string{}
  for _, urlOrFile := range urlsOrFiles {
    if strings.HasPrefix(urlOrFile, "http://") || strings.HasPrefix(urlOrFile, "https://") {
      opts.urls = append(opts.urls, urlOrFile)
    } else {
      // assume file
    }
  }

  Info("Smacking things...")
  counterCh := make(chan chan bool)
  counterDoneCh := make(chan uint64)
  resultCh := make(chan *Result)
  resultDoneCh := make(chan bool)
  urlsCh := make(chan string)
  go counter(&opts, counterCh, counterDoneCh)
  go results(&opts, resultCh, resultDoneCh)
  go urls(&opts, urlsCh)
  beginT := time.Now().UnixNano()
  for i := uint64(0); i < opts.c; i++ {
    go user(&opts, i, counterCh, resultCh, urlsCh)
  }
  <-counterDoneCh
  totalT := time.Now().UnixNano()-beginT
  close(counterCh)
  close(counterDoneCh)
  resultCh <-&Result{done: true}
  <-resultDoneCh
  Info("")
  Info("requsts took %f seconds", (float64(totalT)/1000000000.0))
  Info("%f requests/sec", float64(opts.n)/(float64(totalT)/1000000000.0))
  close(resultCh)
  close(resultDoneCh)
}
