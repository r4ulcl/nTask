
# STEP 1 build executable binary
FROM golang:alpine as builder

# copy files for compile
COPY ./certs $GOPATH/src/github.com/r4ulcl/nTask/certs
COPY ./docs $GOPATH/src/github.com/r4ulcl/nTask/docs
COPY ./globalstructs $GOPATH/src/github.com/r4ulcl/nTask/globalstructs
COPY ./go.mod $GOPATH/src/github.com/r4ulcl/nTask/go.mod
COPY ./go.sum $GOPATH/src/github.com/r4ulcl/nTask/go.sum
COPY ./main.go $GOPATH/src/github.com/r4ulcl/nTask/main.go
COPY ./manager $GOPATH/src/github.com/r4ulcl/nTask/manager
COPY ./worker $GOPATH/src/github.com/r4ulcl/nTask/worker

WORKDIR $GOPATH/src/github.com/r4ulcl/nTask
#get dependancies
#RUN apk -U add alpine-sdk
#RUN go get -d -v
#build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s' -o /go/bin/nTask

#create config folder 
RUN mkdir /config

# STEP 2 build a small image
# start from kali for worker
FROM kalilinux/kali-rolling
#GOPATH doesn-t exists in scratch
ENV GOPATH='/go' 

RUN apt-get update && apt-get install procps net-tools curl nmap -y
RUN apt-get install -y kali-tools-fuzzing 

# Copy our static executable
COPY --from=builder /$GOPATH/bin/nTask /$GOPATH/bin/nTask
# Copy modules
COPY --from=builder $GOPATH/src/github.com/r4ulcl/nTask/worker/modules/ /config/modules/

# Copy swagger
COPY --from=builder $GOPATH/src/github.com/r4ulcl/nTask/docs/ /config/docs/


# Set config folder
WORKDIR  /config

ENTRYPOINT ["/go/bin/nTask"]