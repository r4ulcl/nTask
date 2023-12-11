
# STEP 1 build executable binary
FROM golang:alpine as builder
COPY . $GOPATH/src/github.com/r4ulcl/NetTask

WORKDIR $GOPATH/src/github.com/r4ulcl/NetTask
#get dependancies
#RUN apk -U add alpine-sdk
#RUN go get -d -v
#build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s' -o /go/bin/NetTask

#create config folder 
RUN mkdir /config
RUN cp $GOPATH/src/github.com/r4ulcl/NetTask/sql.sql /config/sql.sql 

# STEP 2 build a small image
# start from kali for worker
FROM kalilinux/kali-rolling
#GOPATH doesn-t exists in scratch
ENV GOPATH='/go' 

# Copy our static executable
COPY --from=builder /$GOPATH/bin/NetTask /$GOPATH/bin/NetTask
#Copy SQL file
COPY --from=builder /config/ /config/
# Copy swagger
COPY --from=builder $GOPATH/src/github.com/r4ulcl/NetTask/docs/ /config/docs/


# Set config folder
WORKDIR  /config

ENTRYPOINT ["/go/bin/NetTask"]