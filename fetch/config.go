// The fetch package is used to retrieve information about Roblox builds.
//
// Several different types of information can be retrieved:
//
//     - Builds: A list of builds, including version hashes.
//     - Latest: Information about the latest build.
//     - APIDump: An API dump for a given hash.
//     - ReflectionMetadata: Reflection metadata for a given hash.
//     - ExplorerIcons: Explorer class icons for a given hash.
//
// The fetch package specializes only in the newer JSON dump format.
package fetch

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxdhist"
	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/xml"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type unzipper struct {
	rc       io.ReadCloser
	deferred []func() error
}

func unzip(rc io.ReadCloser, filename string) (uz *unzipper, err error) {
	uz = &unzipper{}
	type readSeeker interface {
		io.ReaderAt
		io.Seeker
		io.Closer
	}

	// Zip reader requires a ReaderAt and a size.
	var readerAt io.ReaderAt
	var size int64
	switch rs := rc.(type) {
	case readSeeker:
		// Use value as readerAt directly.
		readerAt = rs
		if size, err = rs.Seek(0, io.SeekEnd); err != nil {
			return nil, err
		}
		if _, err = rs.Seek(0, 0); err != nil {
			return nil, err
		}
		uz.Defer(func() error { return rs.Close() })
	default:
		// Write data to temp file, use file as readerAt.
		tmp, err := ioutil.TempFile(os.TempDir(), "")
		if err != nil {
			rc.Close()
			return nil, err
		}
		uz.Defer(func() error { return os.Remove(tmp.Name()) })
		uz.Defer(func() error { return tmp.Close() })
		_, err = io.Copy(tmp, rc)
		rc.Close()
		if err != nil {
			uz.Close()
			return nil, err
		}
		if size, err = tmp.Seek(0, io.SeekEnd); err != nil {
			uz.Close()
			return nil, err
		}
		if _, err = tmp.Seek(0, 0); err != nil {
			uz.Close()
			return nil, err
		}
		readerAt = tmp
	}

	// Unzip data.
	zr, err := zip.NewReader(readerAt, size)
	if err != nil {
		uz.Close()
		return nil, err
	}
	var zfile *zip.File
	for _, zf := range zr.File {
		if zf.Name != filename {
			continue
		}
		zfile = zf
		break
	}
	if zfile == nil {
		uz.Close()
		return nil, errors.New("failed to find file in archive")
	}
	if uz.rc, err = zfile.Open(); err != nil {
		uz.Close()
		return nil, err
	}
	uz.Defer(func() error { return uz.rc.Close() })
	return uz, nil
}

func (uz *unzipper) Defer(fn func() error) {
	uz.deferred = append(uz.deferred, fn)
}

func (uz *unzipper) Read(p []byte) (n int, err error) {
	return uz.rc.Read(p)
}

func (uz *unzipper) Close() (err error) {
	for i := len(uz.deferred) - 1; i >= 0; i-- {
		e := uz.deferred[i]()
		uz.deferred[i] = nil
		if err == nil {
			err = e
		}
	}
	return err
}

func handleGlobalFormat(loc Location, resp io.ReadCloser, err error) (string, io.ReadCloser, error) {
	if err != nil || resp == nil {
		return loc.Format, resp, err
	}
	format := loc.Format
	switch format {
	case ".zip":
		format = path.Ext(loc.URL.Fragment)
		resp, err = unzip(resp, loc.URL.Fragment)
	}
	return format, resp, err
}

func userCacheDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("LocalAppData")
		if dir == "" {
			return "", errors.New("%LocalAppData% is not defined")
		}

	case "darwin":
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Caches"

	case "plan9":
		dir = os.Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib/cache"

	default:
		dir = os.Getenv("XDG_CACHE_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
			}
			dir += "/.cache"
		}
	}

	return dir, nil
}

func expandHash(u url.URL, hash string) (url.URL, error) {
	s := u.String()
	visited := false
	s = os.Expand(s, func(v string) string {
		visited = true
		switch strings.ToLower(v) {
		case "hash":
			return hash
		}
		return ""
	})
	if !visited {
		return u, nil
	}
	v, err := url.Parse(s)
	return *v, err
}

// UnsupportedFormatError indicates that an unsupported format was received.
type UnsupportedFormatError interface {
	error
	// UnsupportedFormat returns the unsupported format.
	UnsupportedFormat() string
}

type errUnsupportedFormat string

func (err errUnsupportedFormat) Error() string {
	return "unsupported format \"" + string(err) + "\""
}

func (err errUnsupportedFormat) UnsupportedFormat() string {
	return string(err)
}

// CacheMode specifies how data is cached between calls.
type CacheMode int

const (
	// Data is never cached.
	CacheNone CacheMode = iota
	// Data is cached in the temporary directory.
	CacheTemp
	// Data is cached in the user cache directory. If unavailable, the
	// temporary directory is used instead.
	CachePerm
	// Data is cached to a custom directory specified by CacheLocation.
	CacheCustom
)

// Location represents where and how a type of data is fetched. See Client.Get
// for how Locations are interpreted.
type Location struct {
	URL    url.URL
	Format string
}

// NewLocation parses a given URL into a Location. The URL is assumed to be
// well-formed. The Format is derived from the extension of the URL path.
func NewLocation(s string) (loc Location) {
	if err := loc.FromString(s); err != nil {
		panic(err)
	}
	return loc
}

// Ext returns the extension of the URL path.
func (loc *Location) Ext() string {
	return path.Ext(loc.URL.Path)
}

// FromString sets the fields of the Location from a URL string.
func (loc *Location) FromString(s string) (err error) {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	loc.URL = *u
	loc.Format = loc.Ext()
	return nil
}

// MarshalJSON implements the json.Marshaller interface. When the Format field
// matches the URL path extension, the Location is written as a JSON string,
// and is otherwise written as a JSON object matching the structure of the
// Location.
func (loc Location) MarshalJSON() (b []byte, err error) {
	if loc.Format == loc.Ext() {
		return json.Marshal(loc.URL.String())
	}
	jurl := struct {
		URL    string
		Format string
	}{
		URL:    loc.URL.String(),
		Format: loc.Format,
	}
	return json.Marshal(jurl)
}

// UnmarshalJSON implements the json.Unmarshaller interface. The JSON value
// can be a string specifying a URL, in which case the Format field is
// determined by the extension of the URL's path. Otherwise, the value must be
// an object that matches the structure of the Location.
func (loc *Location) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		var jurl struct {
			URL    string
			Format string
		}
		if err = json.Unmarshal(b, &jurl); err != nil {
			return err
		}
		if err = loc.FromString(jurl.URL); err != nil {
			return err
		}
		if jurl.Format != "" {
			loc.Format = jurl.Format
		}
		return nil
	}
	return loc.FromString(s)
}

// Config contains locations for each type of data, which consequentially
// specify where and how the data is fetched.
type Config struct {
	Builds,
	Latest,
	APIDump,
	ReflectionMetadata,
	ExplorerIcons Location
}

// Load sets the config from a JSON-formatted stream.
func (cfg *Config) Load(r io.Reader) (err error) {
	return json.NewDecoder(r).Decode(&cfg)
}

// Save writes the config to a stream in JSON format.
func (cfg *Config) Save(w io.Writer) (err error) {
	je := json.NewEncoder(w)
	je.SetEscapeHTML(false)
	je.SetIndent("", "\t")
	return je.Encode(&cfg)
}

// Version represents a Roblox version number.
type Version = rbxdhist.Version

// Build represents information about a single Roblox build.
type Build struct {
	Hash    string
	Date    time.Time
	Version Version
}

// Client is used to perform the fetching of information. It controls where
// data is retrieved from, and how the data is cached.
//
// Each type of information is retrieved by a specific method on a Client.
// Each method reads data in one of several formats, specified by the
// corresponding location. The formats accepted by a particular method are
// described in the documentation for the method.
//
// There are also global formats read by every method. The following global
// formats are available:
//
//     - .zip: The file is a zip archive. A file within this archive is
//       retrieved and read by the method as usual. This file is referred to
//       by the fragment of the URL. For example, the following URL refers to
//       the "file.txt" file: https://example.com/archive.zip#file.txt. The
//       extension of the filename determines the new format.
type Client struct {
	// Config specifies the locations from which data will be retrieved.
	Config Config
	// CacheMode specifies how to cache files.
	CacheMode CacheMode
	// CacheLocation specifies the path to store cached files, when CacheMode
	// is CacheCustom.
	CacheLocation string
	// Client is the HTTP client that performs requests.
	Client *http.Client
}

const cacheDirName = "roblox-fetch"

// Get performs a generic request. The loc argument specifies the address of
// the request. Within the location URL, variables of the form "$var" or
// "${var}" are replaced with the referred value. That said, only the $HASH
// variable (case-insensitive) is replaced with the value of the hash
// argument.
//
// When the URL scheme is "file", the URL is interpreted as a path to a file
// on the file system. In this case, caching is skipped.
//
// Returns the format indicating how the file should be interpreted
// (loc.Format), a ReadCloser that reads the contents of the file, and any
// error the may have occurred.
//
// If loc.Format specifies a global format, it is handled here. In this case,
// the processed format is returned, along with the processed file, which must
// still be closed as usual.
func (client *Client) Get(loc Location, hash string) (format string, rc io.ReadCloser, err error) {
	loc.URL, err = expandHash(loc.URL, hash)
	defer func() {
		format, rc, err = handleGlobalFormat(loc, rc, err)
	}()
	if loc.URL.Scheme == "file" {
		rc, err = os.Open(loc.URL.Path)
		return loc.Format, rc, err
	}
	c := client.Client
	if c == nil {
		c = http.DefaultClient
	}
	var name string
	var f *os.File
	var cacheDir string
	filename := url.PathEscape(loc.URL.Host + loc.URL.Path)
	switch client.CacheMode {
	case CacheTemp:
		cacheDir = filepath.Join(os.TempDir(), cacheDirName)
	case CachePerm:
		dir, err := userCacheDir()
		if err != nil {
			dir = os.TempDir()
		}
		cacheDir = filepath.Join(dir, cacheDirName)
	case CacheCustom:
		cacheDir = client.CacheLocation
	default:
		goto download
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return loc.Format, nil, err
	}
	name = filepath.Join(cacheDir, filename)
	if f, err := os.Open(name); err == nil {
		return loc.Format, f, nil
	}
	f, _ = os.Create(name)
download:
	resp, err := c.Get(loc.URL.String())
	if err != nil {
		return loc.Format, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return loc.Format, nil, errors.New("bad status")
	}
	if f == nil {
		return loc.Format, resp.Body, nil
	}
	_, err = io.Copy(f, resp.Body)
	resp.Body.Close()
	if err != nil {
		f.Close()
		return loc.Format, nil, err
	}
	_, err = f.Seek(0, 0)
	return loc.Format, f, err
}

// Latest returns the latest build, the hash from which can be passed to other
// methods to fetch data corresponding to the latest version. The following
// formats are readable:
//
//     - .json: A single build in JSON format.
//     - (other): A raw stream indicating a version hash. Other build
//       information is empty.
func (client *Client) Latest() (build Build, err error) {
	format, resp, err := client.Get(client.Config.Latest, "")
	if err != nil {
		return build, err
	}
	defer resp.Close()

	switch format {
	case ".json":
		err = json.NewDecoder(resp).Decode(&build)
		return build, err
	default:
		b, err := ioutil.ReadAll(resp)
		if err != nil {
			return build, err
		}
		return Build{Hash: string(b)}, nil
	}
}

// Builds returns a list of builds. The following formats are readable:
//
//     - .txt: A deployment log. Builds from here are filtered and curated to
//       include only those that are interoperable with the fetch package.
//     - .json: A build list in JSON format.
func (client *Client) Builds() (builds []Build, err error) {
	format, resp, err := client.Get(client.Config.Builds, "")
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	switch format {
	case ".json":
		err = json.NewDecoder(resp).Decode(&builds)
		return builds, err
	case ".txt":
		b, err := ioutil.ReadAll(resp)
		if err != nil {
			return nil, err
		}
		stream := rbxdhist.Lex(b)
		pst, _ := time.LoadLocation("America/Los_Angeles")
		// Builds after this date are interoperable.
		epoch := time.Date(2018, 8, 7, 0, 0, 0, 0, pst)
		for i := 0; i < len(stream); i++ {
			switch job := stream[i].(type) {
			case *rbxdhist.Job:
				// Only Studio builds.
				if job.Build != "Studio" || !job.Time.After(epoch) {
					continue
				}
				// Only completed builds.
				if i+1 >= len(stream) {
					continue
				}
				if status, ok := stream[i+1].(*rbxdhist.Status); !ok || *status != "Done" {
					continue
				}
				builds = append(builds, Build{
					Hash:    job.Hash,
					Date:    job.Time,
					Version: job.Version,
				})
			}
		}
		return builds, nil
	}
	return nil, errUnsupportedFormat(format)
}

// APIDump returns the API dump of the given hash. The following formats are
// readable:
//
//     - .json: An API dump in JSON format.
func (client *Client) APIDump(hash string) (root *rbxapijson.Root, err error) {
	format, resp, err := client.Get(client.Config.APIDump, hash)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	switch format {
	case ".json":
		return rbxapijson.Decode(resp)
	}
	return nil, errUnsupportedFormat(format)
}

// ReflectionMetadata returns the reflection metadata for the given hash. The
// following formats are readable:
//
//     - .xml: The RBXMX format.
func (client *Client) ReflectionMetadata(hash string) (root *rbxfile.Root, err error) {
	format, resp, err := client.Get(client.Config.ReflectionMetadata, hash)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	switch format {
	case ".xml":
		return xml.Deserialize(resp, nil)
	}
	return nil, errUnsupportedFormat(format)
}

// readBytes scans until the given delimitor is reached.
func readBytes(r *bufio.Reader, sep []byte) error {
	if len(sep) == 0 {
		return nil
	}
	for {
		if b, err := r.Peek(len(sep)); err != nil {
			return err
		} else if bytes.Equal(b, sep) {
			break
		}
		if _, err := r.Discard(1); err != nil {
			return err
		}
	}
	return nil
}

// ExplorerIcons returns the studio explorer icons for the given hash. The
// following formats are readable:
//
//     - .png: A PNG image.
//     - (other): A PNG embedded within an arbitrary stream of bytes. Because
//       the stream may contain multiple images, the following heuristic is
//       used: the height of the image is 16, the width is a multiple of 16,
//       and is the widest such image.
func (client *Client) ExplorerIcons(hash string) (icons image.Image, err error) {
	format, resp, err := client.Get(client.Config.ExplorerIcons, hash)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	switch format {
	case ".png":
		return png.Decode(resp)
	default:
		header := []byte("\x89PNG\r\n\x1a\n")
		for br := bufio.NewReader(resp); ; {
			if err := readBytes(br, header); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			img, err := png.Decode(br)
			if err != nil || img.Bounds().Dy() != 16 || img.Bounds().Dx()%16 != 0 {
				continue
			}
			if icons == nil || img.Bounds().Dx() > icons.Bounds().Dx() {
				icons = img
			}
		}
		return icons, nil
	}
	return nil, errUnsupportedFormat(format)
}
