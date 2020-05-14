package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/mattn/go-shellwords"
)

var (
	flagAuthCmd = flag.String("e", "", "exec token argument and grab its stdout as token")
	flagToken   = flag.String("t", "", "sourcehut personal token or command if -e is set")
)

type paste struct {
	token   string
	name    string
	content []byte
}

func main() {
	log.SetPrefix("spaste: ")
	log.SetFlags(0)
	flag.Parse()
	flag.Usage = func() {
		log.Print("(-e cmd | -t token) [ files... ]")
		flag.PrintDefaults()
	}

	check := func(err error) {
		if err != nil {
			log.Print(err)
			os.Exit(1)
		}
	}

	token := *flagToken
	cmd := *flagAuthCmd
	if token == "" && cmd == "" {
		log.Print("missing token argument")
		flag.Usage()
		os.Exit(1)
	}
	if cmd != "" {
		t, err := exectoken(cmd)
		check(err)
		token = t
	}

	if flag.NArg() == 0 {
		b, err := ioutil.ReadAll(os.Stdin)
		check(err)
		p := paste{
			token:   token,
			name:    "<stdin>",
			content: b,
		}
		blob, err := spaste(&p)
		check(err)
		fmt.Printf("https://paste.sr.ht/blob/%s\n", blob)
		return
	}

	// TODO(w): join multiple files in single paste?
	for _, file := range flag.Args() {
		b, err := ioutil.ReadFile(file)
		check(err)
		p := paste{
			token:   token,
			name:    "<stdin>",
			content: b,
		}
		blob, err := spaste(&p)
		check(err)
		fmt.Printf("https://paste.sr.ht/blob/%s\n", blob)
	}
}

func tojson(p *paste) ([]byte, error) {
	type postFile struct {
		Name     string `json:"filename"`
		Contents string `json:"contents"`
	}
	type postJSON struct {
		Visibility string     `json:"visibility"`
		Files      []postFile `json:"files"`
	}
	files := []postFile{
		postFile{
			Name:     p.name,
			Contents: string(p.content),
		},
	}
	jsonm := postJSON{
		Visibility: "unlisted",
		Files:      files,
	}

	body, err := json.Marshal(&jsonm)
	if err != nil {
		return nil, fmt.Errorf("tojson: %v", err)
	}
	return body, nil
}

func newRequest(body io.Reader, token string) (*http.Request, error) {
	req, err := http.NewRequest("POST", "https://paste.sr.ht/api/pastes", body)
	if err != nil {
		return nil, fmt.Errorf("newRequest: %v", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func blobfromjson(data []byte) (string, error) {
	type respFile struct {
		Name   string `json:"filename"`
		BlobID string `json:"blob_id"`
	}
	type respJSON struct {
		Files []respFile `json:"files"`
	}
	r := respJSON{}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("blobfromjson: %v", err)
	}
	if len(r.Files) == 0 {
		return "", fmt.Errorf("blobfromjson: unknown data scheme: %s", data)
	}
	// TODO(w): multiple files
	return r.Files[0].BlobID, nil
}

func spaste(p *paste) (string, error) {
	body, err := tojson(p)
	if err != nil {
		return "", err
	}
	req, err := newRequest(bytes.NewReader(body), p.token)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("sourcehut refuse: body: %s", b)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return blobfromjson(b)
}

func exectoken(cmd string) (string, error) {
	if cmd == "" {
		return "", fmt.Errorf("exectoken: empty command")
	}
	sh, err := shellwords.Parse(cmd)
	if err != nil {
		return "", fmt.Errorf("exectoken: %v", err)
	}
	var b bytes.Buffer
	c := exec.Command(sh[0], sh[1:]...)
	c.Stdout = &b
	c.Stderr = os.Stderr
	if err = c.Run(); err != nil {
		return "", fmt.Errorf("exectoken: %v", err)
	}
	return b.String(), nil
}
