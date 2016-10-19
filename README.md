# to build
```
GOOS=linux GOARCH=386 go build -ldflags "-s -w" winner.go
# copy local certificate in container
cp /etc/ssl/certs/ca-certificates.crt ./
docker build -t winner-scratch ./
```

# to run 
```
docker run -p 8000:8000 --rm -e ORGA_ID='XXXXX' -e TOKEN='XXXXXXX' winner-scratch
``` 
