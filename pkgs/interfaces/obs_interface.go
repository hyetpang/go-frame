package interfaces

import (
	"io"
	"net/http"
)

// obs抽象
type OBSInterface interface {
	PutObject(bucketName, objectName string, reader io.Reader) (string, error)
	PutFile(bucketName, objectName, filePath string) (string, error)
	GetSignedUrl(bucket, objectName string, headers map[string]string) (string, http.Header, error)
}
