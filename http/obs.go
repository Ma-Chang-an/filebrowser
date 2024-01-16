package http

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/filebrowser/filebrowser/v2/files"
	obs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

var (
	ak = os.Getenv("AccessKeyID")
	sk = os.Getenv("SecretAccessKey")
	// endpoint填写Bucket对应的Endpoint
	endPoint   = os.Getenv("EndPoint")
	bucketName = os.Getenv("BucketName")
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
		return uploadSingalFile2ObsHandler(file, d)
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
		log.Printf("upload %s", fname)
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
		log.Println("%v", err)
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
	return uploadSingalFile2Obs(fname)
}

func uploadSingalFile2ObsHandler(file *files.FileInfo, d *data) (int, error) {
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
	return uploadSingalFile2Obs(file.Path)
}

func uploadSingalFile2Obs(fname string) (int, error) {
	// 创建obsClient实例
	// 如果使用临时AKSK和SecurityToken访问OBS，需要在创建实例时通过obs.WithSecurityToken方法指定securityToken值。
	obsClient, err := obs.New(ak, sk, endPoint)
	if err != nil {
		log.Printf("Create obsClient error, errMsg: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	defer obsClient.Close()
	input := &obs.PutFileInput{}
	// 指定存储桶名称
	input.Bucket = bucketName
	// 指定上传对象，此处以 example/objectname 为例。
	input.Key = fname[1:]
	// 指定本地文件，此处以localfile为例
	input.SourceFile = fname
	// 文件上传
	output, err := obsClient.PutFile(input)
	if err == nil {
		log.Printf("Put file(%s) under the bucket(%s) successful!\n", input.Key, input.Bucket)
		log.Printf("StorageClass:%s, ETag:%s\n", output.StorageClass, output.ETag)
		return 0, nil
	}
	log.Printf("Put file(%s) under the bucket(%s) fail!\n", input.Key, input.Bucket)
	if obsError, ok := err.(obs.ObsError); ok {
		log.Println("An ObsError was found, which means your request sent to OBS was rejected with an error response.")
		log.Println(obsError.Error())
	} else {
		log.Println("An Exception was found, which means the client encountered an internal problem when attempting to communicate with OBS, for example, the client was unable to access the network.")
		log.Println(err)
	}
	return http.StatusInternalServerError, err
}
