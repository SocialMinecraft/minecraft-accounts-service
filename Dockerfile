FROM golang:1.22

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app ./

CMD ["app"]

#docker build --no-cache -t dmgarvis/minecraft-accounts-service:latest .
#docker push dmgarvis/minecraft-accounts-service:latest

#docker run --env-file .env dmgarvis/minecraft-accounts-service:latest