FROM golang:1.20.4
RUN mkdir /app
RUN mkdir /data
ADD . /app
WORKDIR /app
RUN go build -o main ./app/server.go
EXPOSE 8080
ENTRYPOINT ["/app/main"]