## Yet another socks proxy using gRPC streaming.

### Server

Using docker(using bin file if docker is not supported) to start a server.

```Dockerfile
docker run -itd \
    -p 10465:10465 \
    --name grpc-ss \
    --restart always \
    sdrzlyz/grpc-ss -l :10465
```

### Client

```
client_bin_file -r xxx.xxx.xxx.xxx:10465
```

#### You can download both server and client bin file from github release.