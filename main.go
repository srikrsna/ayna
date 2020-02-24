package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("website name is required")
	}

	root := os.Args[1]
	root = strings.TrimSuffix(root, "/")
	files := make([]string, 0, 100)
	processed := make(map[string]bool)
	files = append(files, root)
	var buf bytes.Buffer

	for len(files) > 0 {
		file := files[0]
		files = files[1:]

		file = strings.TrimPrefix(file, root)
		if processed[file] {
			continue
		}

		processed[file] = true
		res, err := http.Get(root + file)
		if err != nil {
			fmt.Println(file, err)
			continue
			return fmt.Errorf("error trying to download: %s, err: %w", file, err)
		}
		defer res.Body.Close()

		if res.StatusCode-200 > 100 {
			if res.StatusCode == 404 {
				continue
			}
			if res.StatusCode >= 500 {
				fmt.Println(file, res.StatusCode)
				continue
			}
			return fmt.Errorf("download: %v, returned a non success status code: %v", file, res.StatusCode)
		}

		buf.Reset()
		ext := filepath.Ext(file)
		if ext == "" || ext == ".html" || ext == ".htm" {
			doc, err := html.ParseWithOptions(io.TeeReader(res.Body, &buf))
			if err != nil {
				return fmt.Errorf("invalid html returned from: %s", file)
			}

			var f func(n *html.Node)
			f = func(n *html.Node) {
				if n.Type == html.ElementNode {
					switch n.DataAtom {
					case atom.A:
						for _, a := range n.Attr {
							if a.Key == "href" {
								if !strings.HasPrefix(a.Val, "//") && (strings.HasPrefix(a.Val, "/") || strings.HasPrefix(a.Val, root)) {
									files = append(files, cleanUrl(a.Val))
								}
								break
							}
						}
					case atom.Link:
						var (
							f     string
							queue bool
						)
						for _, a := range n.Attr {
							if a.Key == "href" {
								if !strings.HasPrefix(a.Val, "//") && (strings.HasPrefix(a.Val, "/") || strings.HasPrefix(a.Val, root)) {
									f = cleanUrl(a.Val)
								}
							}
							if a.Key == "rel" && a.Val == "stylesheet" {
								queue = true
							}
						}
						if queue && f != "" {
							files = append(files, f)
						}
					case atom.Script, atom.Style, atom.Img, atom.Source, atom.Audio, atom.Video:
						for _, a := range n.Attr {
							if a.Key == "src" || a.Key == "poster" {
								if !strings.HasPrefix(a.Val, "//") && (strings.HasPrefix(a.Val, "/") || strings.HasPrefix(a.Val, root)) {
									files = append(files, cleanUrl(a.Val))
								}
							}

							if a.Key == "srcset" {
								srcs := strings.Split(a.Val, ",")
								for _, src := range srcs {
									splits := strings.Split(src, " ")
									if len(splits) > 0 {
										if !strings.HasPrefix(splits[0], "//") && (strings.HasPrefix(splits[0], "/") || strings.HasPrefix(splits[0], root)) {
											files = append(files, cleanUrl(splits[0]))
										}
									}
								}
							}
						}
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			f(doc)
		} else {
			io.Copy(&buf, res.Body)
		}

		res.Body.Close()

		path := filepath.Join("root", file)
		if ext == "" || ext == ".html" || ext == ".htm" {
			path = filepath.Join(path, "index.html")
		}

		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create a directory, err: %w", err)
		}
		if err := ioutil.WriteFile(path, buf.Bytes(), os.ModePerm); err != nil {
			return fmt.Errorf("unable to write file to disk, err: %w", err)
		}
	}

	return nil
}

func cleanUrl(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return s
	}

	u.RawQuery = ""
	u.Fragment = ""

	return u.String()
}
