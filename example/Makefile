doc:
	swag fmt --dir=./
	swag init --parseDependency --dir=./,./routers/api --quiet=true --ot go
build: doc
	env GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=jsoniter -a -ldflags '-s -w -extldflags "-static"' -o main main.go
	upx -9 -o main-upx main
	mv main-upx main
scp: build
	mv main maxit
	scp maxit akswork001@akswork.plant.test:/home/jorjan/maxit_server
	rm maxit
deploy: build
	scp main akswork001@akswork.plant.test:/home/docker/maxoauth
	ssh akswork.plant.test "sudo docker restart maxoauth"
	rm main
