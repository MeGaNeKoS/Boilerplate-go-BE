package files

// DownloadInput specifies the file to download via path parameter.
type DownloadInput struct {
	Name string `path:"name" example:"sample.txt"`
}
