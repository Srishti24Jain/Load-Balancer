# Load Balancer

# How to use
```bash
 start
go run main.go
```

# Sample curl for register endpoint:
 ```
 $ curl --location --request POST 'localhost:8000/urls/register' \
--header 'Content-Type: application/json' \
--data-raw '{
 "backends": [
  {
   "url": "https://httpstat.us/200"
  },
  {
   "url": "http://duckduckgo.com"
  },
  {
   "url": "http://www.google.com"
  }
 ]

}'
```
# Curl for redirecting through proxy
```  
$ curl --location --request GET 'localhost:8000/proxy' 
  
```