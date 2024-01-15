package http

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/filebrowser/filebrowser/v2/files"
	"github.com/filebrowser/filebrowser/v2/fileutils"
)

var zipHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
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

	if files.IsNamedPipe(file.Mode) {
		setContentDisposition(w, r, file)
		return 0, nil
	}

	if !file.IsDir {
		// return rawFileHandler(w, r, file)
		return zipFileHandler(w, r, file)
	}

	// return rawDirHandler(w, r, d, file)
	return zipDirHandler(w, r, d, file)
})

var unzipHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	// TODO: unzip file
	return http.StatusAccepted, nil
})

func zipDirHandler(w http.ResponseWriter, r *http.Request, d *data, file *files.FileInfo) (int, error) {
	filenames, err := parseQueryFiles(r, file, d.user)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	extension, ar, err := parseQueryAlgorithm(r)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// TODO: create a zip archive in current dir
	// err = ar.Create(w)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }
	defer ar.Close()

	commonDir := fileutils.CommonPrefix(filepath.Separator, filenames...)

	name := filepath.Base(commonDir)
	if name == "." || name == "" || name == string(filepath.Separator) {
		name = file.Name
	}
	// Prefix used to distinguish a filelist generated
	// archive from the full directory archive
	if len(filenames) > 1 {
		name = "_" + name
	}
	name += extension

	// TODO: create a zip archive

	for _, fname := range filenames {
		log.Printf("Failed to archive %s", fname)
		// err = addFile(ar, d, fname, commonDir)
		// if err != nil {
		// 	log.Printf("Failed to archive %s: %v", fname, err)
		// }
	}

	return 0, nil
}

func zipFileHandler(w http.ResponseWriter, r *http.Request, file *files.FileInfo) (int, error) {
	fd, err := file.Fs.Open(file.Path)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer fd.Close()

	// TODO: create a zip archive with only one file
	return 0, nil
}
