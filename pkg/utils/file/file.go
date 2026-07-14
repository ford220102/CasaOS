package file

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"path"
	path2 "path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mholt/archiver/v3"
)

// Default permissions - use more restrictive defaults
const (
	defaultFilePerm  = 0o644 // rw-r--r--
	defaultDirPerm   = 0o755 // rwxr-xr-x
	defaultWritePerm = 0o600 // rw-------
)

// GetSize get the file size
func GetSize(f multipart.File) (int, error) {
	content, err := ioutil.ReadAll(f)
	return len(content), err
}

// GetExt get the file ext
func GetExt(fileName string) string {
	return path.Ext(fileName)
}

// CheckNotExist check if the file exists
func CheckNotExist(src string) bool {
	_, err := os.Stat(src)
	return os.IsNotExist(err)
}

// CheckPermission check if the file has permission
func CheckPermission(src string) bool {
	_, err := os.Stat(src)
	return os.IsPermission(err)
}

// IsNotExistMkDir create a directory if it does not exist
func IsNotExistMkDir(src string) error {
	if notExist := CheckNotExist(src); notExist {
		if err := MkDir(src); err != nil {
			return err
		}
	}
	return nil
}

// MkDir create a directory with safe permissions
func MkDir(src string) error {
	err := os.MkdirAll(src, defaultDirPerm)
	if err != nil {
		return err
	}
	// Only change permissions if it's not a permission issue
	return os.Chmod(src, defaultDirPerm)
}

// RMDir remove a directory
func RMDir(src string) error {
	return os.RemoveAll(src)
}

// Open a file according to a specific mode
func Open(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// MustOpen maximize trying to open the file
func MustOpen(fileName, filePath string) (*os.File, error) {
	src := filePath
	if CheckPermission(src) {
		return nil, fmt.Errorf("file.CheckPermission Permission denied src: %s", src)
	}

	err := IsNotExistMkDir(src)
	if err != nil {
		return nil, fmt.Errorf("file.IsNotExistMkDir src: %s, err: %v", src, err)
	}

	f, err := Open(src+fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, defaultFilePerm)
	if err != nil {
		return nil, fmt.Errorf("Fail to OpenFile :%v", err)
	}

	return f, nil
}

// Exists判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// IsDir判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// IsFile判断所给路径是否为文件
func IsFile(path string) bool {
	return !IsDir(path)
}

// CreateFile creates a new file with safe permissions
func CreateFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, defaultFilePerm)
	if err != nil {
		// If file exists, try with O_TRUNC instead
		if os.IsExist(err) {
			file, err = os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, defaultFilePerm)
			if err != nil {
				return err
			}
			return file.Close()
		}
		return err
	}
	return file.Close()
}

// CreateFileAndWriteContent creates a file and writes content to it
func CreateFileAndWriteContent(path string, content string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultWritePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	
	write := bufio.NewWriter(file)
	if _, err := write.WriteString(content); err != nil {
		return err
	}
	return write.Flush()
}

// IsNotExistCreateFile create a file if it does not exist
func IsNotExistCreateFile(src string) error {
	if notExist := CheckNotExist(src); notExist {
		return CreateFile(src)
	}
	return nil
}

// ReadFullFile reads entire file content
func ReadFullFile(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		return []byte("")
	}
	defer file.Close()
	
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return []byte("")
	}
	return content
}

// copyFileInternal is the internal implementation for copying files
func copyFileInternal(src, dst string, handleExisting func(string) error) error {
	if handleExisting != nil {
		if err := handleExisting(dst); err != nil {
			return err
		}
	}

	srcfd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcfd.Close()

	// Create with safe permissions
	dstfd, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultFilePerm)
	if err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}

	srcinfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	// Only copy permissions if they're safe
	mode := srcinfo.Mode()
	if mode&0o0777 != 0 {
		// Mask out potentially dangerous permissions
		mode = mode &^ 0o7777
		mode |= defaultFilePerm
	}
	return os.Chmod(dst, mode)
}

// handleExistingFile handles existing file based on style
func handleExistingFile(dst, style string) error {
	if Exists(dst) {
		if style == "skip" {
			return errors.New("file exists and skip style selected")
		}
		return os.Remove(dst)
	}
	return nil
}

// CopyFile copies a single file from src to dst
func CopyFile(src, dst, style string) error {
	lastPath := src[strings.LastIndex(src, "/")+1:]
	if !strings.HasSuffix(dst, "/") {
		dst += "/"
	}
	dst += lastPath

	handler := func(d string) error {
		return handleExistingFile(d, style)
	}
	return copyFileInternal(src, dst, handler)
}

// CopySingleFile copies a single file from src to dst with no path modification
func CopySingleFile(src, dst, style string) error {
	handler := func(d string) error {
		return handleExistingFile(d, style)
	}
	return copyFileInternal(src, dst, handler)
}

// GetNoDuplicateFileName checks for duplicate file names
func GetNoDuplicateFileName(fullPath string) string {
	path, fileName := filepath.Split(fullPath)
	fileSuffix := path2.Ext(fileName)
	filenameOnly := strings.TrimSuffix(fileName, fileSuffix)
	for i := 0; Exists(fullPath); i++ {
		fullPath = path2.Join(path, filenameOnly+"("+strconv.Itoa(i+1)+")"+fileSuffix)
	}
	return fullPath
}

// CopyDir copies a whole directory recursively
func CopyDir(src string, dst string, style string) error {
	srcinfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcinfo.IsDir() {
		return CopyFile(src, dst, style)
	}

	lastPath := src[strings.LastIndex(src, "/")+1:]
	dst = dst + "/" + lastPath

	if Exists(dst) {
		if style == "skip" {
			return nil
		}
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
	}

	// Use safe directory permissions
	if err = os.MkdirAll(dst, defaultDirPerm); err != nil {
		return err
	}

	fds, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		if fd.IsDir() {
			if err = CopyDir(srcfp, dst, style); err != nil {
				return err
			}
		} else {
			if err = CopyFile(srcfp, dst, style); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteToPath writes data to a file at the specified path
func WriteToPath(data []byte, path, name string) error {
	fullPath := path
	if strings.HasSuffix(path, "/") {
		fullPath += name
	} else {
		fullPath += "/" + name
	}
	return WriteToFullPath(data, fullPath, defaultWritePerm)
}

// WriteToFullPath writes data to a file with full path
func WriteToFullPath(data []byte, fullPath string, perm fs.FileMode) error {
	if err := IsNotExistCreateFile(fullPath); err != nil {
		return err
	}

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, perm)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.Write(data)
	return err
}

// SpliceFiles concatenates multiple files into one
func SpliceFiles(dir, path string, length int, startPoint int) error {
	fullPath := path

	if err := IsNotExistCreateFile(fullPath); err != nil {
		return err
	}

	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, defaultWritePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	bufferedWriter := bufio.NewWriter(file)

	for i := 0; i < length+startPoint-1; i++ {
		data, err := ioutil.ReadFile(dir + "/" + strconv.Itoa(i+startPoint))
		if err != nil {
			return err
		}
		if _, err := bufferedWriter.Write(data); err != nil {
			return err
		}
	}

	return bufferedWriter.Flush()
}

// GetCompressionAlgorithm returns the appropriate archiver based on format
func GetCompressionAlgorithm(t string) (string, archiver.Writer, error) {
	switch t {
	case "zip", "":
		return ".zip", archiver.NewZip(), nil
	case "tar":
		return ".tar", archiver.NewTar(), nil
	case "targz":
		return ".tar.gz", archiver.NewTarGz(), nil
	case "tarbz2":
		return ".tar.bz2", archiver.NewTarBz2(), nil
	case "tarxz":
		return ".tar.xz", archiver.NewTarXz(), nil
	case "tarlz4":
		return ".tar.lz4", archiver.NewTarLz4(), nil
	case "tarsz":
		return ".tar.sz", archiver.NewTarSz(), nil
	default:
		return "", nil, errors.New("format not implemented")
	}
}

// AddFile adds a file to the archiver
func AddFile(ar archiver.Writer, path, commonPath string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() && !info.Mode().IsRegular() {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if path != commonPath {
		filename := strings.TrimPrefix(path, commonPath)
		filename = strings.TrimPrefix(filename, string(filepath.Separator))
		err = ar.Write(archiver.File{
			FileInfo: archiver.FileInfo{
				FileInfo:   info,
				CustomName: filename,
			},
			ReadCloser: file,
		})
		if err != nil {
			return err
		}
	}

	if info.IsDir() {
		names, err := file.Readdirnames(0)
		if err != nil {
			return err
		}

		for _, name := range names {
			if err := AddFile(ar, filepath.Join(path, name), commonPath); err != nil {
				log.Printf("Failed to archive %v", err)
			}
		}
	}

	return nil
}

// CommonPrefix returns the common prefix of multiple paths
func CommonPrefix(sep byte, paths ...string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return path.Clean(paths[0])
	}

	c := []byte(path.Clean(paths[0]))
	c = append(c, sep)

	for _, v := range paths[1:] {
		v = path.Clean(v) + string(sep)

		if len(v) < len(c) {
			c = c[:len(v)]
		}
		for i := 0; i < len(c); i++ {
			if v[i] != c[i] {
				c = c[:i]
				break
			}
		}
	}

	for i := len(c) - 1; i >= 0; i-- {
		if c[i] == sep {
			c = c[:i]
			break
		}
	}

	return string(c)
}

// GetFileOrDirSize returns the size of a file or directory
func GetFileOrDirSize(path string) (int64, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if fileInfo.IsDir() {
		return DirSizeB(path + "/")
	}
	return fileInfo.Size(), nil
}

// DirSizeB returns the size of a directory in bytes
func DirSizeB(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// MoveFile moves a file from source to destination
func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	defer inputFile.Close()

	outputFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultWritePerm)
	if err != nil {
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()

	if _, err := io.Copy(outputFile, inputFile); err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}

	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}

// ReadLine reads a specific line from a file
func ReadLine(lineNumber int, path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	lineCount := 1
	for fileScanner.Scan() {
		if lineCount == lineNumber {
			return fileScanner.Text()
		}
		lineCount++
	}
	return ""
}

// NameAccumulation generates a unique filename in a directory
func NameAccumulation(name string, dir string) string {
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return name
	}

	base := name
	index := strings.LastIndex(base, "_")
	if index < 0 {
		index = len(base)
	}

	for i := 1; ; i++ {
		newPath := filepath.Join(dir, fmt.Sprintf("%s_%d", base[:index], i))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return fmt.Sprintf("%s_%d", base[:index], i)
		}
	}
}

// ParseFileHeader parses multipart file headers
func ParseFileHeader(h []byte, boundary []byte) (map[string]string, bool) {
	arr := bytes.Split(h, boundary)
	result := make(map[string]string)

	for _, item := range arr {
		tarr := bytes.Split(item, []byte(";"))
		if len(tarr) != 2 {
			continue
		}

		tbyte := tarr[1]
		tbyte = bytes.ReplaceAll(tbyte, []byte("\r\n--"), []byte(""))
		tbyte = bytes.ReplaceAll(tbyte, []byte("name=\""), []byte(""))
		tempArr := bytes.Split(tbyte, []byte("\"\r\n\r\n"))
		if len(tempArr) != 2 {
			continue
		}
		result[strings.TrimSpace(string(tempArr[0]))] = strings.TrimSpace(string(tempArr[1]))
	}
	return result, true
}

// ReadToBoundary reads data until a boundary is found
func ReadToBoundary(boundary []byte, stream io.ReadCloser, target io.WriteCloser) ([]byte, bool, error) {
	readData := make([]byte, 1024*8)
	readDataLen := 0
	buf := make([]byte, 1024*4)
	bLen := len(boundary)
	reachEnd := false

	for !reachEnd {
		readLen, err := stream.Read(buf)
		if err != nil {
			if err != io.EOF || readLen <= 0 {
				return nil, true, err
			}
			reachEnd = true
		}

		copy(readData[readDataLen:], buf[:readLen])
		readDataLen += readLen
		if readDataLen < bLen+4 {
			continue
		}

		loc := bytes.Index(readData[:readDataLen], boundary)
		if loc >= 0 {
			target.Write(readData[:loc-4])
			return readData[loc:readDataLen], reachEnd, nil
		}

		target.Write(readData[:readDataLen-bLen-4])
		copy(readData[0:], readData[readDataLen-bLen-4:])
		readDataLen = bLen + 4
	}

	target.Write(readData[:readDataLen])
	return nil, reachEnd, nil
}

// ParseFromHead parses data from the beginning of a stream
func ParseFromHead(readData []byte, readTotal int, boundary []byte, stream io.ReadCloser) (map[string]string, []byte, error) {
	buf := make([]byte, 1024*8)
	foundBoundary := false
	boundaryLoc := -1

	for {
		readLen, err := stream.Read(buf)
		if err != nil {
			if err != io.EOF {
				return nil, nil, err
			}
			break
		}

		if readTotal+readLen > cap(readData) {
			return nil, nil, fmt.Errorf("not found boundary")
		}

		copy(readData[readTotal:], buf[:readLen])
		readTotal += readLen

		if !foundBoundary {
			boundaryLoc = bytes.LastIndex(readData[:readTotal], boundary)
			if boundaryLoc == -1 {
				continue
			}
			foundBoundary = true
		}

		startLoc := boundaryLoc + len(boundary)
		fileHeadLoc := bytes.Index(readData[startLoc:readTotal], []byte("\r\n\r\n"))
		if fileHeadLoc == -1 {
			continue
		}

		fileHeadLoc += startLoc
		headMap, ret := ParseFileHeader(readData, boundary)
		if !ret {
			return headMap, nil, fmt.Errorf("ParseFileHeader fail:%s", string(readData[startLoc:fileHeadLoc]))
		}
		return headMap, readData[fileHeadLoc+4 : readTotal], nil
	}
	return nil, nil, fmt.Errorf("reach to stream EOF")
}
