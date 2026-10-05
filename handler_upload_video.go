package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	fmt.Println("uploading video", videoID, "for user", userID)

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Failed to get video metadata", err)
		return
	}

	const maxMemory = 1 << 30 // 1GB
	videoFile, videoHeaders, err := r.FormFile("video")

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to get video file", err)
		return
	}
	defer videoFile.Close()

	mediaType, _, err := mime.ParseMediaType(videoHeaders.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse media type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", nil)
		return
	}

	tempFile, err := os.CreateTemp(filepath.Join(cfg.assetsRoot, "/", "videos"), "tubely-upload.*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, videoFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to copy video file", err)
		return
	}

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to seek to start of temp file", err)
		return
	}

	processedPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}
	defer os.Remove(processedPath)

	processedFile, err := os.Open(processedPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to open processed file", err)
		return
	}
	defer processedFile.Close()

	aspectRatio, err := getVideoAspectRatio(processedPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get video aspect ratio", err)
		return
	}
	var key string
	switch {
	case aspectRatio == "16:9":
		key = fmt.Sprintf("landscape/%s", videoIDString)
	case aspectRatio == "9:16":
		key = fmt.Sprintf("portrait/%s", videoIDString)
	default:
		key = fmt.Sprintf("other/%s", videoIDString)
	}

	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(key),
		Body:        processedFile,
		ContentType: aws.String(mediaType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload video to S3", err)
		return
	}

	// videoMetadata.VideoURL = aws.String(fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key))
	videoMetadata.VideoURL = aws.String(fmt.Sprintf("%s,%s", cfg.s3Bucket, key))
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video metadata", err)
		return
	}
}

func (cfg *apiConfig) generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	preSignClient := s3.NewPresignClient(s3Client)
	params := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	presignedURL, err := preSignClient.PresignGetObject(context.Background(), &params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return presignedURL.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	split := strings.Split(*video.VideoURL, ",")
	if len(split) != 2 {
		return video, fmt.Errorf("invalid video URL: %s", video.VideoURL)
	}
	bucket, key := split[0], split[1]
	presignedURL, err := cfg.generatePresignedURL(cfg.s3Client, bucket, key, 5*time.Minute)
	if err != nil {
		return video, err
	}
	video.VideoURL = &presignedURL
	return video, nil
}
