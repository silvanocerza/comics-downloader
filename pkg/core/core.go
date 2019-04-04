package core

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/BakeRolls/mri"
	"github.com/Girbons/comics-downloader/pkg/util"
	epub "github.com/bmaupin/go-epub"
	"github.com/jung-kurt/gofpdf"
	"github.com/mholt/archiver"
	"github.com/schollz/progressbar"
	log "github.com/sirupsen/logrus"
)

// manga output format supported
const (
	CBR  = "cbr"
	CBZ  = "cbz"
	EPUB = "epub"
	PDF  = "pdf"
)

// Comic struct contains all the informations about a comic
type Comic struct {
	Author      string
	Name        string
	IssueNumber string
	Source      string
	URLSource   string
	Links       []string
	Options     map[string]string
	Format      string
}

// SetAuthor sets the comic author
func (c *Comic) SetAuthor(author string) {
	c.Author = author
}

// SetName sets the comic name
func (c *Comic) SetName(name string) {
	c.Name = name
}

// SetIssueNumber sets the comic issue number
func (c *Comic) SetIssueNumber(issueNumber string) {
	c.IssueNumber = issueNumber
}

// SetURLSource sets the URL Source
func (c *Comic) SetURLSource(URLSource string) {
	c.URLSource = URLSource
}

// SetSource sets the source without the http prefix
func (c *Comic) SetSource(source string) {
	c.Source = source
}

// SetLinks sets the image links retrieved for a manga
func (c *Comic) SetImageLinks(links []string) {
	c.Links = links
}

// SetFormat sets the comic output format
func (c *Comic) SetFormat(format string) {
	switch strings.ToLower(format) {
	case EPUB:
		c.Format = EPUB
	case CBR:
		c.Format = CBR
	case CBZ:
		c.Format = CBZ
	default:
		c.Format = PDF
	}
}

// SetInfo will sets the name, issueNumber
func (c *Comic) SetInfo(name, issueNumber string) {
	c.Name = name
	c.IssueNumber = issueNumber
}

// SplitURL return the url splitted by "/"
func (c *Comic) SplitURL() []string {
	return strings.Split(c.URLSource, "/")
}

// SetOptions set options to the current comic
func (c *Comic) SetOptions(options map[string]string) {
	c.Options = options
}

// generateFileName will return the path where the file is should be saved
func (c *Comic) generateFileName(dir string) string {
	return fmt.Sprintf("%s/%s.%s", dir, c.IssueNumber, c.Format)
}

// RetrieveImageFromResponse will return the image byte and its type
func (c *Comic) retrieveImageFromResponse(response *http.Response) (io.Reader, string) {
	var (
		content io.Reader
		tp      string
	)

	switch c.Source {
	case "mangarock.com":
		// mangarock image needs to be decoded first
		img, decErr := mri.Decode(response.Body)
		if decErr != nil {
			log.WithFields(log.Fields{
				"error":  decErr,
				"source": c.Source,
			}).Error("Image decode failed")
		}

		imgData := new(bytes.Buffer)
		if err := util.ConvertTo8BitPNG(img, imgData); err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"source": c.Source,
			}).Error("Error trying to convert the image")
		}

		content = imgData
		tp = "png"
	default:
		content = response.Body
		tp = util.ImageType(response.Header["Content-Type"][0])
	}

	return content, tp

}

// makeEPUB create the epub file
func (c *Comic) makeEPUB() {
	var err error
	// used to check if the epub cover already exists
	isCoverSet := false
	// used to add the image in the epub section
	imgTag := `<img src="%s" alt="Cover Image" />`
	// setup a new Epub instance
	e := epub.NewEpub(c.IssueNumber)
	// set Epub title
	e.SetTitle(fmt.Sprintf("%s-%s", c.Name, c.IssueNumber))
	// check if the author exists for this comic
	if c.Author != "" {
		e.SetAuthor(c.Author)
	}
	// in order to create an epub we'll need to download all the images so we create a tempdir for that
	tempDir, err := ioutil.TempDir("", "comics-images")
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"source": c.Source,
		}).Fatal("There was a problem creating the temp directory")
	}
	defer os.RemoveAll(tempDir) // clean up

	if err = os.Chdir(tempDir); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was a problem creating the temp directory")
	}
	// setup the progress bar
	bar := progressbar.New(len(c.Links))
	// this will show up the progress bar since the beginning
	if barErr := bar.RenderBlank(); barErr != nil {
		log.WithFields(log.Fields{
			"error": barErr,
		}).Error("There was a problem while rendering the progressbar")
	}

	for i, link := range c.Links {
		if link != "" {
			rsp, err := http.Get(link)
			if err == nil {
				defer rsp.Body.Close()
				// retrieve the image from the response
				content, tp := c.retrieveImageFromResponse(rsp)
				// create a tempfile to store the image
				tmpfile, err := ioutil.TempFile(tempDir, fmt.Sprintf("image.*.%s", tp))
				defer os.Remove(tmpfile.Name()) // clean up

				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Fatal("Unable to create tempfile")
				}

				if _, err = io.Copy(tmpfile, content); err != nil {
					log.WithFields(log.Fields{
						"error":  err,
						"url":    link,
						"source": c.Source,
					}).Error("Error while copying content to tempfile")
				}
				// add the image to the epub will return a path
				imgpath, err := e.AddImage(tmpfile.Name(), "")
				if err != nil {
					log.WithFields(log.Fields{
						"error":  err,
						"source": c.Source,
					}).Error("Can't add image")
				}
				// if the cover is not set we'll use the first image
				// otherwise the image will be added as a section
				if !isCoverSet {
					isCoverSet = true
					e.SetCover(imgpath, "")
				} else {
					_, err = e.AddSection(fmt.Sprintf(imgTag, imgpath), "", "", "")
					if err != nil {
						log.WithFields(log.Fields{
							"error":  err,
							"source": c.Source,
						}).Error("Can't add section ")
					}
				}
			} else {
				log.WithFields(log.Fields{
					"error":  err,
					"url":    link,
					"source": c.Source,
				}).Error("Something went wrong with the current url")
			}
		}
		if barErr := bar.Add(i); barErr != nil {
			log.WithFields(log.Fields{
				"error": barErr,
			}).Error("There was problem while increasing the progressbar")
		}
	}
	if err = os.Chdir(tempDir); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error while trying to change directory")
	}
	// Set progressbar to its maximum
	if err = bar.Finish(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Cannot set the progressbar to its maximum")
	}
	// get the PathSetup where the file should be saved
	// e.g. /www.mangarock.com/comic-name/
	dir, err := util.PathSetup(c.Source, c.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was a problem while creating the manga path")
	}

	if err = e.Write(c.generateFileName(dir)); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was an error creating the epub file")
	} else {
		log.Info("EPUB correctly saved")
	}
}

// makePDF create the pdf file
func (c *Comic) makePDF() {
	var err error
	// setup the pdf
	pdf := gofpdf.New("P", "mm", "A4", "")
	// setup the progress bar
	bar := progressbar.New(len(c.Links))
	// show up the progress bar since the beginning
	if barErr := bar.RenderBlank(); barErr != nil {
		log.WithFields(log.Fields{
			"error": barErr,
		}).Error("There was a problem while rendering the progressbar")
	}
	// for each link get the image to add to the pdf file
	for i, link := range c.Links {
		if link != "" {
			rsp, err := http.Get(link)
			if err == nil {
				defer rsp.Body.Close()
				// add a new PDF page
				pdf.AddPage()
				content, tp := c.retrieveImageFromResponse(rsp)
				// The image is directly added to the pdf without being saved to the disk
				imageOptions := gofpdf.ImageOptions{ImageType: tp, ReadDpi: false, AllowNegativePosition: true}
				pdf.RegisterImageOptionsReader(link, imageOptions, content)
				// set the image position on the pdf page
				pdf.Image(link, 0, 0, 210, 0, false, tp, 0, "")
				// increase the progressbar
			} else {
				log.WithFields(log.Fields{
					"source": c.URLSource,
					"url":    link,
					"error":  err,
				}).Error("Something went wrong with the current url")
				pdf.SetError(err)
			}
		}
		if barErr := bar.Add(i); barErr != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("There was problem while increasing the progressbar")
		}
	}
	// Set progressbar to its maximum
	if err = bar.Finish(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Cannot set the progressbar to its maximum")
	}
	// get the PathSetup where the file should be saved
	// e.g. /www.mangarock.com/comic-name/
	dir, err := util.PathSetup(c.Source, c.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was a problem while creating the manga path")
	}

	// Save the pdf file
	if err = pdf.OutputFileAndClose(c.generateFileName(dir)); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was an error while making the PDF")
	}

	if pdf.Ok() {
		log.Info("pdf file correctly saved")
	}
}

// makeCBRZ will create the CBR/CBZ
func (c *Comic) makeCBRZ() {
	var filesToAdd []string
	var err error
	// setup a new Epub instance
	archive := archiver.NewZip()
	// in order to create the archive we'll need to download all the images
	tempDir, err := ioutil.TempDir("", "comics-images")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was a problem creating the temp directory")
	}

	defer os.RemoveAll(tempDir) // clean up
	if err = os.Chdir(tempDir); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was a problem while trying to change directory")
	}
	// setup the progress bar
	bar := progressbar.New(len(c.Links))
	// show up the progress bar since the beginning
	if barErr := bar.RenderBlank(); barErr != nil {
		log.WithFields(log.Fields{
			"error": barErr,
		}).Error("There was a problem while rendering the progressbar")
	}

	for i, link := range c.Links {
		if link != "" {
			rsp, err := http.Get(link)
			if err == nil {
				defer rsp.Body.Close()
				// retrieve the image from the response
				content, tp := c.retrieveImageFromResponse(rsp)
				// create a tempfile to store the image
				tmpfile, err := ioutil.TempFile(tempDir, fmt.Sprintf("%d-image.*.%s", i, tp))
				defer os.Remove(tmpfile.Name()) // clean up

				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Fatal("Unable to create tempfile")
				}

				if _, err = io.Copy(tmpfile, content); err != nil {
					log.WithFields(log.Fields{
						"error":  err,
						"url":    link,
						"source": c.Source,
					}).Error("Error while copying content to tempfile")
				}

				filesToAdd = append(filesToAdd, tmpfile.Name())

			} else {
				log.WithFields(log.Fields{
					"url":    link,
					"source": c.Source,
				}).Error("Something went wrong with the current url", err)
			}
		}

		if barErr := bar.Add(i); barErr != nil {
			log.Warning("There was problem while increasing the progressbar")
		}
	}
	// Set progressbar to its maximum
	if err = bar.Finish(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Cannot set the progressbar to its maximum")
	}

	if err = os.Chdir(tempDir); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error while trying to change directory")
	}
	// e.g. /www.mangarock.com/comic-name/
	dir, err := util.PathSetup(c.Source, c.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"source": c.Source,
			"error":  err,
		}).Fatal("There was a problem while creating the manga path")
	}
	// the archive must be created as .zip
	// then we can change the extension to .cbr or .cbz
	zipArchiveName := fmt.Sprintf("%s/%s.zip", dir, c.IssueNumber)
	newName := fmt.Sprintf("%s/%s.%s", dir, c.IssueNumber, c.Format)
	if err = archive.Archive(filesToAdd, zipArchiveName); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("There was an error creating the archive")
	} else {
		if err := os.Rename(zipArchiveName, newName); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("There was a problem while trying to rename the archive")
		}
		log.Info("file correctly saved")
	}
}

// MakeComic will create the file based on the output format selected.
func (c *Comic) MakeComic() {
	switch c.Format {
	case EPUB:
		c.makeEPUB()
	case CBR, CBZ:
		c.makeCBRZ()
	default:
		c.makePDF()
	}
}
