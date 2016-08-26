package pistis

import (
	"github.com/go-martini/martini"
	"net/http"
	"github.com/martini-contrib/render"
	"github.com/cayleygraph/cayley"
	"io/ioutil"
	"github.com/boltdb/bolt"
	"fmt"
	"strings"
	"encoding/json"
	"github.com/cayleygraph/cayley/quad"
)

var (
	cdnStore *bolt.DB
)

func initCDNStore() error {
	cdnStore, err := bolt.Open("cdn.db", 0600, nil)
	if err!=nil {
		log.Fatal(err)
		return err
	}

	cdnStore.Batch(func (tx *bolt.Tx)error{
		tx.CreateBucketIfNotExists([]byte("cdn"))
		return nil
	})

	return nil
}

func CDNServer(r martini.Router) {
	r.Group("/cdn", func(r martini.Router) {
		r.Post("", upload)
		r.Get("/:user/:fname", download)
		r.Delete("/:user/:fname",remove)
	})
}

type uploadResult struct {
	url           string
	thumbnail_url string
	name          string
	size          int
	delete_url    string
	delete_type   string
}

func upload(r render.Render, req *http.Request) {
	req.ParseMultipartForm(1024 * 1024)
	images := req.MultipartForm.File["image"]
	if images == nil || len(images) < 1 {
		r.Text(http.StatusNoContent, "no image data found")
		return
	}

	fh := images[0]
	filename := fh.Filename
	contentType := fh.Header["Content-Type"][0]
	ftype := strings.Split(contentType, "/")
	fname := filename[0:len(filename) - len(ftype)]
	//username:=req.Header["username"][0]
	user := "bino"

	numberedFilename := func(n int) string {
		if n == 0 {
			return fmt.Sprintf("/%s/%s", user, filename)
		}else if len(ftype) == 0 {
			return fmt.Sprintf("/%s/%s(%d)", user, fname, n)
		}else {
			return fmt.Sprintf("/%s/%s(%d).%s", user, fname, n, ftype)
		}
	}

	fmt.Println(filename, fname, contentType, fh, user)

	check := func(id string) bool {
		p := cayley.StartPath(store, quad.StringToValue("image")).Out(id)
		it := p.BuildIterator()
		defer it.Close()
		return it.Next()
	}
	id := numberedFilename(0)
	for n := 0; check(id); n = n + 1 {
		id = numberedFilename(n)
	}

	size := 0
	if f, err := fh.Open(); err != nil {
		panic(err)
		r.Error(http.StatusInternalServerError)
	}else {
		if data, err := ioutil.ReadAll(f); err != nil {
			panic(err)
			r.Error(http.StatusInternalServerError)
		}else {
			size=len(data)
			if err := cdnStore.Batch(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("cdn"))
				return b.Put([]byte(id), data)
			}); err != nil {
				panic(err)
				r.Error(http.StatusInternalServerError)
			}
		}
	}
	url := fmt.Sprintf("%s/cdn%s", httpServerURL, id)
	uploadRes := uploadResult{
		url :url,
		thumbnail_url :url,
		name :filename,
		size:size,
		delete_url:url,
		delete_type:"DELETE",
	}
	if rjson, err := json.Marshal(uploadRes);err!=nil{
		panic(err)
	}else{
		r.Text(http.StatusOK, fmt.Sprintf("{\"files\":[%s]",rjson))
	}
}

func download(params martini.Params, req *http.Request, r render.Render) {
	user := params["user"]
	fname := params["fname"]
	id:=fmt.Sprintf("/%s/%s",user,fname)
	var data []byte
	cdnStore.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("cdn"))
		data = b.Get([]byte(id))
		return nil
	})
	r.Data(http.StatusOK, data)
}

func remove(params martini.Params, req *http.Request, r render.Render) {
	user := params["user"]
	fname := params["fname"]
	id:=fmt.Sprintf("/%s/%s",user,fname)
	var data []byte
	cdnStore.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("cdn"))
		return b.Delete([]byte(id))
	})
	r.Data(http.StatusOK, data)
}

