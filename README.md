# ayna

A simple tool to mirror any website.

## Usage

```sh
go get github.com/srikrsna/ayna
ayna https://example.com
```

It clones the website in the root folder.

### Known Issues

Does not download images specified in CSS files. Eg: Background Images like 
```css
body {
  background-image: url('https://example.com/background.png')
}
```
