package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/Girbons/comics-downloader/pkg/detector"
	"github.com/Girbons/comics-downloader/pkg/loader"
	log "github.com/sirupsen/logrus"
)

func Run(link, format, country string) {
	// link is required
	if link == "" {
		fmt.Println("url parameter is required")
		os.Exit(1)
	}

	if !strings.HasSuffix(link, ",") {
		link = link + ","
	}

	// check if the format is supported
	if !detector.DetectFormatOutput(format) {
		log.WithFields(log.Fields{
			"format": format,
		}).Error("Format not supported pdf will be used instead")
	}

	for _, u := range strings.Split(link, ",") {
		if u != "" {
			// check if the link is supported
			source, check := detector.DetectComic(u)
			if !check {
				log.WithFields(log.Fields{"site": u}).Error("This site is not supported :(")
				continue
			}

			log.WithFields(log.Fields{"link": u}).Info("Downloading...")
			// in case the link is supported
			// setup the right strategy to parse a comic
			comic, err := loader.LoadComicFromSource(source, u, country)
			if err != nil {
				log.WithFields(log.Fields{"link": u, "error": err}).Error("Unable to load the site strategy...")
				continue
			}
			comic.SetFormat(format)
			comic.MakeComic()
		}
	}
}
