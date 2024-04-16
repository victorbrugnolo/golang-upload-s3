package main

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
)

var (
	s3Client *s3.S3
	s3Bucket string
	wg       sync.WaitGroup
)

func init() {
	godotenv.Load()

	err := godotenv.Load("../../.env")
	if err != nil {
		panic("Error loading .env file")
	}

	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String("us-east-1"),
			Credentials: credentials.NewStaticCredentials(
				os.Getenv("AWS_ACCESS_KEY"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"),
				"",
			),
		},
	)

	if err != nil {
		panic(err)
	}

	s3Client = s3.New(sess)
	s3Bucket = "golang-upload-tmp-files"
}

func main() {
	dir, err := os.Open("../../tmp")

	if err != nil {
		panic(err)
	}

	defer dir.Close()

	uploadControl := make(chan struct{}, 100)

	for {
		files, err := dir.ReadDir(1)

		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Printf("Error reading directory: %s\n", err)
			continue
		}

		wg.Add(1)
		uploadControl <- struct{}{}
		go uploadFile(files[0].Name(), uploadControl)
	}

	wg.Wait()
}

func uploadFile(filename string, uploadControl <-chan struct{}) {
	defer wg.Done()

	completeFileName := fmt.Sprintf("../../tmp/%s", filename)
	fmt.Printf("Uploading file %s to bucket %s\n", completeFileName, s3Bucket)

	f, err := os.Open(completeFileName)

	if err != nil {
		fmt.Printf("Error opening file %s\n", completeFileName)
		<-uploadControl
		return
	}

	defer f.Close()

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(filename),
		Body:   f,
	})

	if err != nil {
		fmt.Printf("Error uploading file %s\n", completeFileName)
		<-uploadControl
		return
	}

	fmt.Printf("File %s uploaded successfully\n", completeFileName)
	<-uploadControl
}
