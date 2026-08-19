package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20 // 10MB
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}
	defer file.Close()
	
	mediaType := header.Header.Get("Content-Type")	
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "No media type", nil)
		return
	}

	imageData, err := io.ReadAll(file)
	videoMetadata, err := cfg.db.GetVideo(videoID)

	if userID != videoMetadata.UserID {
		respondWithError(w, http.StatusUnauthorized, "You are unauthorized", nil)
		return
	}

	base64Data := base64.StdEncoding.EncodeToString(imageData)
	dataUrl := fmt.Sprintf("data:%v;base64,%v",mediaType, base64Data)
	videoMetadata.ThumbnailURL = &dataUrl

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w,http.StatusInternalServerError,"Failed to updated video",err)
		return
	}
	
	respondWithJSON(w, http.StatusOK, videoMetadata)
}
