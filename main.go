package main

import (
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

const (
	port       = 8081
	storageDir = "storage"
)

var pageTemplate = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Файловый сервер</title>
<style>
body {
  font-family: system-ui, sans-serif;
  background: #f6f8fa;
  color: #222;
  margin: 0;
}
header {
  background: #0078d7;
  color: white;
  padding: 1rem;
  text-align: center;
}
main {
  max-width: 900px;
  margin: 2rem auto;
  background: white;
  border-radius: 16px;
  padding: 2rem;
  box-shadow: 0 4px 20px rgba(0,0,0,0.1);
}
input[type=file] {
  margin: 1rem 0;
}
button {
  background: #0078d7;
  border: none;
  color: white;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  cursor: pointer;
}
button:hover {
  background: #005fa3;
}
ul { list-style: none; padding: 0; }
li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  border-bottom: 1px solid #eee;
}
.file-link { text-decoration: none; color: #0078d7; }
.file-link:hover { text-decoration: underline; }
</style>
</head>
<body>
<header>
  <h1>Файловый сервер</h1>
</header>
<main>
  <h2>Загрузка файлов или папок</h2>
  <form action="/upload" method="post" enctype="multipart/form-data">
    <input type="file" name="files" id="files" webkitdirectory directory multiple>
    <button type="submit">Загрузить</button>
  </form>

  <h2>Файлы и папки</h2>
  <ul>
    {{range .}}
    <li>
      <a class="file-link" href="/files/{{.Path}}" download>{{.Name}}</a>
      <form style="display:inline" action="/delete" method="post">
        <input type="hidden" name="path" value="{{.Path}}">
        <button type="submit">Удалить</button>
      </form>
    </li>
    {{else}}
    <li><em>Папка пуста</em></li>
    {{end}}
  </ul>
</main>
</body>
</html>
`))

type FileInfo struct {
	Name string
	Path string
}

func main() {
	os.MkdirAll(storageDir, 0755)

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(storageDir))))

	printLocalAddresses(port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func printLocalAddresses(port int) {
	ifaces, _ := net.Interfaces()
	fmt.Printf("\nServer start:\n")
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.IsPrivate() {
				fmt.Printf("  http://%s:%d\n", ip.String(), port)
			}
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var files []FileInfo
	filepath.Walk(storageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(storageDir, path)
		files = append(files, FileInfo{Name: rel, Path: rel})
		return nil
	})
	pageTemplate.Execute(w, files)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(1 << 30)
	files := r.MultipartForm.File["files"]
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()
		targetPath := filepath.Join(storageDir, filepath.FromSlash(fileHeader.Filename))
		os.MkdirAll(filepath.Dir(targetPath), 0755)
		out, err := os.Create(targetPath)
		if err != nil {
			continue
		}
		io.Copy(out, file)
		out.Close()
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	path := filepath.Join(storageDir, r.FormValue("path"))
	os.Remove(path)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
