package files_test

import (
	"io/ioutil"
	"os"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/files/backends/fs"

	//. "github.com/theduke/go-appkit/files"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backend implementations", func() {

	var backend kit.ApiFileBackend

	createFile := func(bucket, id string, content []byte) {
		writer, err := backend.WriterById(bucket, id)
		Expect(err).ToNot(HaveOccurred())

		_, err2 := writer.Write(content)
		Expect(err2).ToNot(HaveOccurred())
		Expect(writer.Flush()).ToNot(HaveOccurred())
		//Expect(writer.Close()).ToNot(HaveOccurred())

		Expect(backend.HasFileById(bucket, id)).To(BeTrue())
	}

	testBackend := func() {
		It("Should set name", func() {
			backend.SetName("test")
			Expect(backend.Name()).To(Equal("test"))
		})

		It("Should create bucket", func() {
			err := backend.CreateBucket("test", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(backend.HasBucket("test")).To(BeTrue())
			Expect(backend.Buckets()).To(Equal([]string{"test"}))
		})

		It("Should delete bucket", func() {
			err := backend.CreateBucket("test", nil)
			Expect(err).ToNot(HaveOccurred())

			err = backend.DeleteBucket("test")
			Expect(err).ToNot(HaveOccurred())

			Expect(backend.HasBucket("test")).To(BeFalse())
			Expect(backend.Buckets()).To(Equal([]string{}))
		})

		It("Should allow to create file", func() {
			err := backend.CreateBucket("test", nil)
			Expect(err).ToNot(HaveOccurred())

			createFile("test", "testfile", []byte("test content"))
		})

		It("Should return all file ids", func() {
			err := backend.CreateBucket("test", nil)
			Expect(err).ToNot(HaveOccurred())

			createFile("test", "testfile1", []byte("test content"))
			createFile("test", "testfile2", []byte("test content"))
			Expect(backend.FileIDs("test")).To(Equal([]string{"testfile1", "testfile2"}))
		})

		It("Should allow to read file", func() {
			// Create bucket first.
			err := backend.CreateBucket("test", nil)
			Expect(err).ToNot(HaveOccurred())

			// Create a file.
			content := []byte("test content")
			createFile("test", "testfile", content)

			// Confirm that the file can be read.
			reader, err := backend.ReaderById("test", "testfile")
			Expect(err).ToNot(HaveOccurred())

			data, err2 := ioutil.ReadAll(reader)
			Expect(err2).ToNot(HaveOccurred())
			//reader.Close()

			Expect(data).To(Equal(content))				
		})

		It("Should clear bucket", func() {
			// Create bucket first.
			err := backend.CreateBucket("test", nil)
			Expect(err).ToNot(HaveOccurred())

			content := []byte("test content")
			createFile("test", "testfile", content)
			createFile("test", "testfile2", content)

			Expect(backend.ClearBucket("test")).ToNot(HaveOccurred())
			Expect(backend.HasFileById("test", "testfile")).To(BeFalse())
			Expect(backend.HasFileById("test", "testfile2")).To(BeFalse())
			Expect(backend.FileIDs("test")).To(Equal([]string{}))
		})

		It("Should delete file", func() {
			// Create bucket first.
			err := backend.CreateBucket("test", nil)
			Expect(err).ToNot(HaveOccurred())

			content := []byte("test content")
			createFile("test", "testfile", content)

			Expect(backend.DeleteFileById("test", "testfile")).ToNot(HaveOccurred())
			Expect(backend.HasFileById("test", "testfile")).To(BeFalse())
		})
	}

	Context("fs Backend", func() {
		BeforeEach(func() {
			err := os.RemoveAll("tmp")
			Expect(err).ToNot(HaveOccurred())

			var err2 error
			backend, err2 = fs.New("tmp")
			Expect(err2).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll("tmp")
		})

		testBackend()
	})

})
