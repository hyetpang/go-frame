package interfaces

import (
	"io"
	"net/http"
)

// obs抽象
type OBSInterface interface {
	PutObject(bucketName, objectName string, reader io.Reader) (string, error)
	PutFile(bucketName, objectName, filePath string) (string, error)
	GetSignedUrl(bucket, objectName string, isPublicRead bool) (string, http.Header, error)
}
