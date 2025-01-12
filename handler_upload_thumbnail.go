package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	ct := header.Header.Get("Content-Type")
	fmt.Printf("content type: %v\n", ct)
	//imageData, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "unable to read image data", err)
	// 	return
	// }

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to get video metadata", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "user id does not match video owner", err)
		return
	}

	var extension string
	switch ct {
	case "image/png":
		extension = "png"
	case "image/jpeg", "image/jpg":
		extension = "jpg"
	default:
		extension = "txt"
	}

	//encodedString := base64.StdEncoding.EncodeToString(imageData)
	fileText := fmt.Sprintf("%s.%s", videoIDString, extension)
	filePath := filepath.Join(cfg.assetsRoot, fileText)

	fmt.Printf("File Path: %s", filePath)
	f, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error creating the file", err)
		return
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "error copying the file", err)
		return
	}

	//fileUrl := fmt.Sprintf("data:text/plain;base64,%v", encodedString)
	// videoThumbStruct := thumbnail{
	// 	data:      imageData,
	// 	mediaType: ct,
	// }

	// videoThumbnails[videoID] = videoThumbStruct

	url := fmt.Sprintf("http://localhost:%d/assets/%s", 8091, fileText)
	fmt.Printf("URL: %s", url)

	video.ThumbnailURL = &url

	cfg.db.UpdateVideo(video)
	respondWithJSON(w, http.StatusOK, video)
}
