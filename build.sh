docker run -it --name frontend -v ${pwd}:/mnt/data node:20.9.0-alpine /bin/sh
cd /mnt/data/frontend && npm config set registry https://mirrors.huaweicloud.com/repository/npm/ && npm ci && npm run build && exit
docker run -it --name backend -v ${pwd}:/mnt/data golang:1.20-alpine /bin/sh
go env -w GO111MODULE=on && go env -w  GOPROXY=https://goproxy.cn,direct && cd /mnt/data/ && go build && exit
docker build -t filebrowser-test ./