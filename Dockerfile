FROM golang:latest

COPY . /app/

WORKDIR /app

COPY cert.pem key.pem /app/

ENTRYPOINT ["go", "run", "wachinbot.go"]
