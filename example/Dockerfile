FROM alpine
ADD main ./
ADD ./conf/app.toml ./conf/
# 解决时区和http证书的问题
RUN echo "http://mirrors.aliyun.com/alpine/v3.4/main/" > /etc/apk/repositories && apk update && apk add --no-cache ca-certificates && update-ca-certificates && apk --no-cache add tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone
EXPOSE 8061
CMD ["./main"]
