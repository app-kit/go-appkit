	BackendID() string
	SetBackendID(string) error

	Title() string
	SetTitle(string)

	Description() string
	SetDescription(string)

	Writer(create bool) (string, *bufio.Writer, ApiError)
	Writer(f ApiFile, create bool) (string, *bufio.Writer, ApiError)
	WriterById(bucket, id string, create bool) (string, *bufio.Writer, ApiError)
	SetApp(*App)

	AddBackend(ApiFileBackend)
	// Taken a file in the file system, gather information about it,
	// store it in the default backend and return a file modelfile in the file system, gather information about it,
	// store it in the default backend and return a file model
	BuildFile(bucket, filePath string, user ApiUser) (ApiFile, ApiError)

	BuildFileInBackend(backend, bucket, filePath string, user ApiUser) (ApiFile, ApiError)
