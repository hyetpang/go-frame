package interfaces

import "io"

type OBSInterface interface {
	PutObject(bucketName, objectName string, reader io.Reader) (string, error)
	PutFile(bucketName, objectName, filePath string) (string, error)
}
