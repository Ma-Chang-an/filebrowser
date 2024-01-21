package http

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/filebrowser/filebrowser/v2/files"
	obs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

var obsHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	if !d.user.Perm.Create {
		return http.StatusAccepted, nil
	}

	file, err := files.NewFileInfo(files.FileOptions{
		Fs:         d.user.Fs,
		Path:       r.URL.Path,
		Modify:     d.user.Perm.Modify,
		Expand:     false,
		ReadHeader: d.server.TypeDetectionByHeader,
		Checker:    d,
	})
	if err != nil {
		return errToStatus(err), err
	}

	if !file.IsDir {
		// return rawFileHandler(w, r, file)
		return uploadSignalFile2ObsHandler(file, d)
	}

	// return rawDirHandler(w, r, d, file)
	return uploadDir2ObsHandler(r, d, file)
})

func uploadDir2ObsHandler(r *http.Request, d *data, file *files.FileInfo) (int, error) {
	filenames, err := parseQueryFiles(r, file, d.user)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	for _, fname := range filenames {
		uploadDir2Obs(r, d, fname)
	}

	return 0, nil
}

func uploadDir2Obs(r *http.Request, d *data, fname string) (int, error) {
	if !d.Check(fname) {
		log.Println("not allowed operation")
		return http.StatusInternalServerError, fmt.Errorf("not allowed operation")
	}
	info, err := d.user.Fs.Stat(fname)
	if err != nil {
		log.Printf("%v", err)
		return http.StatusInternalServerError, err
	}

	if !info.IsDir() && !info.Mode().IsRegular() {
		return 0, nil
	}
	if info.IsDir() {
		file, err := d.user.Fs.Open(fname)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		defer file.Close()
		names, err := file.Readdirnames(0)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		for _, name := range names {
			fPath := filepath.Join(fname, name)
			uploadDir2Obs(r, d, fPath)
		}
		return 0, nil
	}
	return uploadSignalFile2Obs(d, fname)
}

func uploadSignalFile2ObsHandler(file *files.FileInfo, d *data) (int, error) {
	if !d.Check(file.Path) {
		return http.StatusInternalServerError, fmt.Errorf("not allowed operation")
	}
	info, err := d.user.Fs.Stat(file.Path)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if !info.IsDir() && !info.Mode().IsRegular() {
		return 0, nil
	}
	return uploadSignalFile2Obs(d, file.Path)
}

func uploadSignalFile2Obs(d *data, fname string) (int, error) {
	if d.user.ObsInfo.AccessKeyId == "" || d.user.ObsInfo.SecretAccessKey == "" || d.user.ObsInfo.BucketName == "" || d.user.ObsInfo.EndPoint == "" {
		return http.StatusBadRequest, fmt.Errorf("need set environment variable: AccessKeyID, SecretAccessKey, EndPoint, BucketName")
	}
	obsClient, err := obs.New(d.user.ObsInfo.AccessKeyId, d.user.ObsInfo.SecretAccessKey, d.user.ObsInfo.EndPoint)
	if err != nil {
		log.Printf("Create obsClient error, errMsg: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	defer obsClient.Close()
	input := &obs.PutFileInput{}
	input.Bucket = d.user.ObsInfo.BucketName
	input.Key = fname[1:]
	input.SourceFile = filepath.Join(d.server.Root, fname)

	output, err := obsClient.PutFile(input)
	if err == nil {
		log.Printf("Put file(%s) under the bucket(%s) successful!StorageClass:%s, ETag:%s\n",
			input.Key, input.Bucket, output.StorageClass, output.ETag)
		return 0, nil
	}
	log.Printf("Put file(%s) under the bucket(%s) fail!\n", input.Key, input.Bucket)
	if obsError, ok := err.(obs.ObsError); ok {
		log.Printf("An ObsError was found, which means your request sent to OBS was rejected with an error response. %v\n", obsError.Error())
	} else {
		log.Printf("An Exception was found, which means the client encountered an internal problem when attempting to communicate with OBS, for example, the client was unable to access the network.%v\n", err)
	}
	return http.StatusInternalServerError, err
}
