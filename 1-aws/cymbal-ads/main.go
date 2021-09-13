package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	bucketName     = ""
	uploadFilename = "adUpload.html"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveAd)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	bucketName = os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		port = "my-awesome-bucket"
	}

	if err := initializeAds(); err != nil {
		fmt.Printf("Could not write ad UI to S3: %v", err)
		os.Exit(1)
	}
	log.Printf("🔈 Cymbal Ads listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// Get from S3 bucket
// https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/s3/s3_download_object.go
func serveAd(w http.ResponseWriter, r *http.Request) {
	log.Printf("Get ad request: %s", r.URL.Path)

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	file, err := os.Create("adDownload.html")
	if err != nil {
		fmt.Printf("Unable to open file %v", err)
	}

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(uploadFilename),
		})
	if err != nil {
		fmt.Printf("Unable to download item %q, %v", uploadFilename, err)
	}

	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")

	// Serve downloaded HTML file
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, file.Name())
}

// Write to S3 bucket
func initializeAds() error {
	log.Println("Uploading adUpload.html to the S3 bucket")

	// Read HTML file into memory
	file, err := os.Open(uploadFilename)
	if err != nil {
		return fmt.Errorf("unable to open file - %v", err)
	}
	defer file.Close()

	// Create S3 session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return err
	}

	// Upload to S3
	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(uploadFilename),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("unable to upload %q to %q, %v", uploadFilename, bucketName, err)
	}

	fmt.Printf("Successfully uploaded %q to %q\n", uploadFilename, bucketName)
	return nil
}
