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
      "url": "http://www.thecocktaildb.com/api/json/v1/1/search.php?s=margarita"
    },
    {
      "url": "http://www.reddit.com/r/Wallstreetbets/top.json?limit=10&t=year"
    },
    {
      "url": "http://www.7timer.info/bin/api.pl?lon=113.17&lat=23.09&product=astro&output=xml"
    }
  ]
}'

```
# Curl for redirecting through proxy
```  
$ curl --location --request GET 'localhost:8000/proxy' 
  
```
