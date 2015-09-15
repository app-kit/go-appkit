
	. "github.com/theduke/go-appkit/error"
	Reader() (io.ReadCloser, Error)
	Writer(create bool) (string, io.WriteCloser, Error)
	Buckets() ([]string, Error)
	HasBucket(string) (bool, Error)
	CreateBucket(string, ApiBucketConfig) Error
	ConfigureBucket(string, ApiBucketConfig) Error
	ClearBucket(bucket string) Error
	DeleteBucket(bucket string) Error
	ClearAll() Error
	FileIDs(bucket string) ([]string, Error)
	HasFile(ApiFile) (bool, Error)
	HasFileById(bucket, id string) (bool, Error)
	DeleteFile(ApiFile) Error
	DeleteFileById(bucket, id string) Error
	Reader(ApiFile) (io.ReadCloser, Error)
	ReaderById(bucket, id string) (io.ReadCloser, Error)
	Writer(f ApiFile, create bool) (string, io.WriteCloser, Error)
	WriterById(bucket, id string, create bool) (string, io.WriteCloser, Error)
	BuildFile(file ApiFile, user ApiUser, filePath string, deleteDir bool) Error
	FindOne(id string) (ApiFile, Error)
	Find(*db.Query) ([]ApiFile, Error)
	Create(ApiFile, ApiUser) Error
	Update(ApiFile, ApiUser) Error
	Delete(ApiFile, ApiUser) Error