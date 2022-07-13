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
   "url": "https://www.7timer.info/bin/astro.php?lon=113.2&lat=23.1&ac=0&unit=metric&output=json&tzshift=0"
  },
  {
   "url": "https://www.7timer.info/bin/api.pl?lon=113.17&lat=23.09&product=astro&output=xml"
  },
  {
   "url": "http://www.7timer.info/bin/astro.php?lon=113.17&lat=23.09&ac=0&lang=en&unit=metric&output=internal&tzshift=0"
  }
 ]
}'

```
# Curl for redirecting through proxy
```  
$ curl --location --request GET 'localhost:8000/proxy' 
  
```