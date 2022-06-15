package interfaces

import "io"

// obs抽象
type OBSInterface interface {
	PutObject(bucketName, objectName string, reader io.Reader) (string, error)
	PutFile(bucketName, objectName, filePath string) (string, error)
}
