smack
=====

Web Server Benchmark Tool

```
smack [options] (url|file)+
(url|file)+: a space separated list of urls and/or files containing a newline separated list of urls
options:
 -n: (uint var) the total number of smacks
 -c: (uint var) the number of users smacking (concurrency)
 -r: (bool flag) if specified, will pick a random url from those specified for each request
 -v: (bool flag) if specified, will output more information. useful for debugging but hinders performance
 -p: (bool flag) if specified, smack will panic if an error (not bad http status) occurs while trying to request a url
e.g. "smack -n 10000 -c 100 -r /tmp/urls.txt" will smack the urls in /tmp/urls.txt randomly for a total number of 10000 requests with 100 users smacking at a time.
```
