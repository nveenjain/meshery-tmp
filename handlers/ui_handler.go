package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// ServeUI - helps serve static files for both meshery ui and provider ui
func ServeUI(w http.ResponseWriter, r *http.Request, reqBasePath, baseFolderPath string) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	reqURL := r.URL.EscapedPath()
	logrus.Debugf(reqURL)
	reqURL = strings.Replace(reqURL, reqBasePath, "", 1)
	var filePath strings.Builder
	filePath.WriteString(reqURL)
	if reqURL == "/" || reqURL == "" {
		filePath.WriteString("index.html")
	} else if filepath.Ext(r.URL.EscapedPath()) == "" {
		filePath.WriteString(".html")
	}
	finalPath := filepath.Join(baseFolderPath, filePath.String())
	logrus.Debugf(finalPath)
	http.ServeFile(w, r, finalPath)
}
