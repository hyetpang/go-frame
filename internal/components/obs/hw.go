package obs

import (
	"io"
	"net/http"

	"github.com/HyetPang/go-frame/internal/components/obs/hw"
	"github.com/HyetPang/go-frame/pkgs/interfaces"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/HyetPang/go-frame/pkgs/validate"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func NewHw() interfaces.OBSInterface {
	conf := new(config)
	err := viper.UnmarshalKey("obs", conf)
	if err != nil {
		logs.Fatal("获取obs配置出错", zap.Error(err))
	}
	validate.Must(conf)
	client, err := hw.New(conf.AK, conf.SK, conf.Endpoint)
	if err != nil {
		logs.Fatal("构建华为obs客户端对象出错", zap.Error(err))
	}
	return &hwClient{ObsClient: client, config: conf}
}

type hwClient struct {
	*hw.ObsClient
	*config
}

func (hc *hwClient) PutObject(bucketName, objectName string, reader io.Reader) (string, error) {
	input := &hw.PutObjectInput{}
	input.Bucket = bucketName
	input.Key = objectName
	input.Body = reader
	_, err := hc.ObsClient.PutObject(input)
	if err != nil {
		if obsError, ok := err.(hw.ObsError); ok {
			logs.Error("华为OBS上传文件出错", zap.Error(err), zap.Any("req", input), zap.Int("status_code", obsError.StatusCode), zap.String("message", obsError.Message))
		} else {
			logs.Error("华为OBS上传文件出错", zap.Error(err), zap.Any("req", input))
		}
		return "", err
	}
	return "https://" + bucketName + "." + hc.config.Endpoint + "/" + objectName, nil
}

func (hc *hwClient) PutFile(bucketName, objectName, filePath string) (string, error) {
	input := &hw.PutFileInput{}
	input.Bucket = bucketName
	input.Key = objectName
	input.SourceFile = filePath
	_, err := hc.ObsClient.PutFile(input)
	if err != nil {
		if obsError, ok := err.(hw.ObsError); ok {
			logs.Error("华为OBS上传文件出错", zap.Error(err), zap.Any("req", input), zap.Int("status_code", obsError.StatusCode), zap.String("message", obsError.Message))
		} else {
			logs.Error("华为OBS上传文件出错", zap.Error(err), zap.Any("req", input))
		}
		return "", err
	}
	return "https://" + bucketName + "." + hc.config.Endpoint + "/" + objectName, nil
}

func (hc *hwClient) PutFileWithSigned(signedUrl string, actualSignedRequestHeaders http.Header, sourceFile string) (output *hw.PutObjectOutput, err error) {
	return hc.ObsClient.PutFileWithSignedUrl(signedUrl, actualSignedRequestHeaders, sourceFile)
}

func (hc *hwClient) GetSignedUrl(bucket, objectName string, headers map[string]string) (string, http.Header, error) {
	putObjectInput := &hw.CreateSignedUrlInput{Headers: headers}
	putObjectInput.Method = hw.HttpMethodPut
	putObjectInput.Bucket = bucket
	putObjectInput.Key = objectName
	putObjectInput.Expires = 3600
	rsp, err := hc.ObsClient.CreateSignedUrl(putObjectInput)
	if err != nil {
		logs.Error("获取华为obs签名的url出错", zap.Error(err))
		return "", nil, err
	}
	return rsp.SignedUrl, rsp.ActualSignedRequestHeaders, nil
}
