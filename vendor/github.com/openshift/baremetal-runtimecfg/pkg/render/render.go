package render

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
)

const ext = ".tmpl"

var extLen = len(ext)

var log = logrus.New()

func RenderFile(renderPath, templatePath string, cfg interface{}) error {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.WithFields(logrus.Fields{
			"path": templatePath,
		}).Error("Failed to parse template")
		return err
	}

	renderFile, err := os.Create(renderPath)
	if err != nil {
		log.WithFields(logrus.Fields{
			"path": renderPath,
		}).Error("Failed to create file")
		return err
	}
	defer renderFile.Close()

	// Make sure we propagate any special permissions
	templateStat, err := os.Stat(templatePath)
	if err != nil {
		log.WithFields(logrus.Fields{
			"path": templatePath,
		}).Error("Failed to stat template")
		return err
	}
	err = os.Chmod(renderPath, templateStat.Mode())
	if err != nil {
		log.WithFields(logrus.Fields{
			"path": renderPath,
		}).Error("Failed to set permissions on file")
		return err
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, cfg)
	if err != nil {
		log.WithFields(logrus.Fields{
			"path": renderPath,
		}).Error("Failed to render template")
		return err
	}
	// The string we get back is a single line with \n's. For readability,
	// split it and write it line-by-line.
	lines := strings.Split(buf.String(), "\n")
	for _, line := range lines {
		log.Info(line)
	}

	log.WithFields(logrus.Fields{
		"path": renderPath,
	}).Info("Runtimecfg rendering template")
	return tmpl.Execute(renderFile, cfg)
}

func Render(outDir string, paths []string, cfg interface{}) error {
	tempPaths := paths
	if len(paths) == 1 {
		fi, err := os.Stat(paths[0])
		if err != nil {
			log.WithFields(logrus.Fields{
				"path": paths[0],
			}).Error("Failed to stat file")
		}
		if fi.Mode().IsDir() {
			templateDir := paths[0]
			files, err := ioutil.ReadDir(templateDir)
			if err != nil {
				log.WithFields(logrus.Fields{
					"path": templateDir,
				}).Error("Failed to read template directory")
				return err
			}
			tempPaths = make([]string, 0)
			for _, entryFi := range files {
				if entryFi.Mode().IsRegular() {
					if path.Ext(entryFi.Name()) == ext {
						tempPaths = append(tempPaths, path.Join(templateDir, entryFi.Name()))
					}
				}
			}
		}
	}
	for _, templatePath := range tempPaths {
		if path.Ext(templatePath) != ext {
			return fmt.Errorf("Template %s does not have the right extension. Must be '%s'", templatePath, ext)
		}

		baseName := path.Base(templatePath)
		renderPath := path.Join(outDir, baseName[:len(baseName)-extLen])
		err := RenderFile(renderPath, templatePath, cfg)
		if err != nil {
			log.WithFields(logrus.Fields{
				"path": templatePath,
				"err":  err,
			}).Error("Failed to render template")
		}
	}
	return nil
}
