package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// Box : Reads in JWT and authenticates.
type Box struct {
	configFile  string
	accessToken string
}

// BoxJWTRequest : Basic structure for a Box API JWT.
type BoxJWTRequest struct {
	BoxAppSettings struct {
		ClientID     string `json:"clientID"`
		ClientSecret string `json:"clientSecret"`
		AppAuth      struct {
			PublicKeyID string `json:"publicKeyID"`
			PrivateKey  string `json:"privateKey"`
			Passphrase  string `json:"passphrase"`
		} `json:"appAuth"`
	} `json:"boxAppSettings"`
	EnterpriseID string `json:"enterpriseID"`
}

// AccessResponse : Object returned by a successful request to the Box API.
type AccessResponse struct {
	AccessToken  string        `json:"access_token"`
	ExpiresIn    int           `json:"expires_in"`
	RestrictedTo []interface{} `json:"restricted_to"`
	TokenType    string        `json:"token_type"`
}

// FileObject : A Box File object.
type FileObject struct {
	Type           string         `json:"type"`
	ID             string         `json:"id"`
	FileVersion    FileVersion    `json:"file_version"`
	SequenceID     string         `json:"sequence_id"`
	Etag           string         `json:"etag"`
	Sha1           string         `json:"sha1"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Size           int            `json:"size"`
	PathCollection PathCollection `json:"path_collection"`
	CreatedAt      string         `json:"created_at"`
	ModifiedAt     string         `json:"modified_at"`
	CreatedBy      User           `json:"created_by"`
	ModifiedBy     User           `json:"modified_by"`
	OwnedBy        User           `json:"owned_by"`
	SharedLink     SharedLink     `json:"shared_link"`
	Parent         Parent         `json:"parent"`
	ItemStatus     string         `json:"item_status"`
}

// FolderObject : A Box Folder object.
type FolderObject struct {
	Type              string            `json:"type"`
	ID                string            `json:"id"`
	SequenceID        string            `json:"sequence_id"`
	Etag              string            `json:"etag"`
	Name              string            `json:"name"`
	CreatedAt         string            `json:"created_at"`
	ModifiedAt        string            `json:"modified_at"`
	Description       string            `json:"description"`
	Size              int               `json:"size"`
	PathCollection    PathCollection    `json:"path_collection,omitempty"`
	CreatedBy         User              `json:"created_by,omitempty"`
	ModifiedBy        User              `json:"modified_by,omitempty"`
	OwnedBy           User              `json:"owned_by,omitempty"`
	SharedLink        SharedLink        `json:"shared_link,omitempty"`
	FolderUploadEmail FolderUploadEmail `json:"folder_upload_email,omitempty"`
	Parent            Parent            `json:"parent,omitempty"`
	ItemStatus        string            `json:"item_status"`
	ItemCollection    ItemCollection    `json:"item_collection,omitempty"`
	Tags              []string          `json:"tags"`
}

// FileVersion : Contains version information of a FileObject.
type FileVersion struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Sha1 string `json:"sha1"`
}

// Entries : A more in-depth response containing more information about box objects.
type Entries struct {
	EntriesMini
	Sha1              string         `json:"sha1 "`
	Description       string         `json:"description"`
	Size              int            `json:"size"`
	PathCollection    PathCollection `json:"path_collection,omitempty"`
	CreatedAt         string         `json:"created_at"`
	ModifiedAt        string         `json:"modified_at"`
	TrashedAt         interface{}    `json:"trashed_at,omitempty"`
	PurgedAt          interface{}    `json:"purged_at,omitempty"`
	ContentCreatedAt  string         `json:"content_created_at"`
	ContentModifiedAt string         `json:"content_modified_at"`
	CreatedBy         User           `json:"created_by,omitempty"`
	ModifiedBy        User           `json:"modified_by,omitempty"`
	OwnedBy           User           `json:"owned_by,omitempty"`
	SharedLink        SharedLink     `json:"shared_link,omitempty"`
	Parent            Parent         `json:"parent,omitempty"`
	ItemStatus        string         `json:"item_status"`
}

// EntriesMini : Basic structure for response carrying info about box objects.
type EntriesMini struct {
	Type       string      `json:"type"`
	ID         string      `json:"id"`
	SequenceID interface{} `json:"sequence_id,omitempty"`
	Etag       string      `json:"etag,omitempty"`
	Name       string      `json:"name"`
}

// PathCollection : The total amount of entries in a given path, as well as the entries themselves.
type PathCollection struct {
	TotalCount int       `json:"total_count"`
	Entries    []Entries `json:"entries"`
}

// User : Contains information about a Box user.
type User struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Login string `json:"login"`
}

// Permissions : Flags for downloaded files.
type Permissions struct {
	CanDownload bool `json:"can_download"`
	CanPreview  bool `json:"can_preview"`
}

// SharedLink : A shared link to a downloadable file.
type SharedLink struct {
	URL               string      `json:"url"`
	DownloadURL       interface{} `json:"download_url,omitempty"`
	VanityURL         interface{} `json:"vanity_url,omitempty"`
	IsPasswordEnabled bool        `json:"is_password_enabled"`
	UnsharedAt        interface{} `json:"unshared_at,omitempty"`
	DownloadCount     int         `json:"download_count"`
	PreviewCount      int         `json:"preview_count"`
	Access            string      `json:"access"`
	Permissions       Permissions `json:"permissions,omitempty"`
}

// FolderUploadEmail : Access level and email address of upload folder.
type FolderUploadEmail struct {
	Access string `json:"access"`
	Email  string `json:"email"`
}

// Parent : Parent folder of a returned box object.
type Parent struct {
	Type       string      `json:"type"`
	ID         string      `json:"id"`
	SequenceID interface{} `json:"sequence_id,omitempty"`
	Etag       interface{} `json:"etag,omitempty"`
	Name       string      `json:"name"`
}

// ItemCollection : Total count up to the limit of the number of entries in a folder, as well as the entries themselves.
type ItemCollection struct {
	TotalCount int           `json:"total_count"`
	Entries    []EntriesMini `json:"entries"`
	Offset     int           `json:"offset"`
	Limit      int           `json:"limit"`
}

// Order : Defines how to sort objects.
type Order struct {
	By        string `json:"by"`
	Direction string `json:"direction"`
}

// EmbeddedFile : An HTML embeddable file.
type EmbeddedFile struct {
	Type              string `json:"type"`
	ID                string `json:"id"`
	Etag              string `json:"etag"`
	ExpiringEmbedLink struct {
		URL string `json:"url"`
	} `json:"expiring_embed_link"`
}

// NewBox : Creates a new server authenticator.
func NewBox(file string) *Box {
	box := &Box{file, ""}
	os.Setenv("authURL", "https://api.box.com/oauth2/token")
	return box
}

// HTTPRequest : Runs an HTTP request via a defined method.
func (box *Box) HTTPRequest(method string, url string, payload io.Reader, headers map[string]string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	} else {
		if len(headers) == 0 {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if box.accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+box.accessToken)
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	respBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	if response.StatusCode != 200 {
		log.Println(" >> URL    :", url)
		log.Println(" >> Status :", response.Status)
	}
	return respBytes, nil
}

// RequestAccessToken : Get valid ACCESS_TOKEN using JWT.
func (box *Box) RequestAccessToken() error {
	name, err := ioutil.ReadFile(box.configFile)
	var boxConfig BoxJWTRequest

	err = json.Unmarshal(name, &boxConfig)

	if err != nil {
		log.Println(err)
		return err
	}

	// Create a unique 32 character long string.
	rBytes := make([]byte, 32)
	_, err = rand.Read(rBytes)
	if err != nil {
		log.Println(err)
		return err
	}
	jti := base64.URLEncoding.EncodeToString(rBytes)

	// Build the header. This includes the PublicKey as the ID.
	token := jwt.New(jwt.SigningMethodRS512)
	token.Header["keyid"] = boxConfig.BoxAppSettings.AppAuth.PublicKeyID

	// Construct claims.
	claims := token.Claims.(jwt.MapClaims)
	claims["iss"] = boxConfig.BoxAppSettings.ClientID
	claims["sub"] = boxConfig.EnterpriseID
	claims["box_sub_type"] = "enterprise"
	claims["aud"] = os.Getenv("authURL")
	claims["jti"] = jti
	claims["exp"] = time.Now().Add(time.Second * 10).Unix()

	// Decrypt the PrivateKey using its passphrase.
	signedKey, err := jwt.ParseRSAPrivateKeyFromPEMWithPassword(
		[]byte(boxConfig.BoxAppSettings.AppAuth.PrivateKey),
		boxConfig.BoxAppSettings.AppAuth.Passphrase,
	)

	if err != nil {
		log.Println(err)
		return err
	}

	// Build the assertion from the signedKey and claims.
	assertion, err := token.SignedString(signedKey)

	if err != nil {
		log.Println(err)
		return err
	}

	// Build the access token request.
	payload := url.Values{}
	payload.Add("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	payload.Add("assertion", assertion)
	payload.Add("client_id", boxConfig.BoxAppSettings.ClientID)
	payload.Add("client_secret", boxConfig.BoxAppSettings.ClientSecret)

	// Post the request to the Box API.
	response, err := box.HTTPRequest("POST", os.Getenv("authURL"), bytes.NewBufferString(payload.Encode()), nil)
	if err != nil {
		log.Println(err)
		return err
	}

	// Set the access token.
	var ar AccessResponse
	err = json.Unmarshal(response, &ar)
	if err != nil {
		log.Println(err)
		return err
	}
	box.accessToken = ar.AccessToken

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// File Functions

// UploadFile : Creates an Access Token to the Box API, then uploads a given name to the specified folder.
func (box *Box) UploadFile(file interface{}, newName string, folderID string) (*PathCollection, error) {
	box.RequestAccessToken()

	t := reflect.TypeOf(file)

	var name string
	if t.Name() == "string" {
		name = file.(string)
	} else {
		name = newName
	}
	if newName == "" && name != "" {
		newName = name
	}

	var contents []byte
	if t.Name() == "" {
		contents = file.([]byte)
	} else {
		f, err := os.Open(name)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()

		contents, err = ioutil.ReadAll(f)
		if err != nil {
			log.Println(err)
		}
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		log.Println(err)
	}
	part.Write(contents)

	err = writer.WriteField("filename", name)
	if err != nil {
		log.Println(err)
	}

	err = writer.Close()
	if err != nil {
		log.Println(err)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = writer.FormDataContentType()
	headers["Content-Length"] = string(body.Len())

	response, err := box.HTTPRequest("POST",
		"https://upload.box.com/api/2.0/files/content?attributes={%22name%22:%22"+newName+"%22,%20%22parent%22:{%22id%22:%22"+folderID+"%22}}",
		body, headers)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fileObject := &PathCollection{}
	json.Unmarshal(response, &fileObject)

	return fileObject, nil
}

// GetFileInfo : Returns information about the file with 'ID' fileID.
func (box *Box) GetFileInfo(fileID string) (*FileObject, error) {
	box.RequestAccessToken()
	response, err := box.HTTPRequest("GET", "https://api.box.com/2.0/files/"+fileID, nil, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fileObject := &FileObject{}
	json.Unmarshal(response, &fileObject)

	return fileObject, nil
}

// GetEmbedLink : Returns information about the file with 'ID' fileID.
func (box *Box) GetEmbedLink(fileID string) (*EmbeddedFile, error) {
	box.RequestAccessToken()
	response, err := box.HTTPRequest("GET", "https://api.box.com/2.0/files/"+fileID+"?fields=expiring_embed_link", nil, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fileObject := &EmbeddedFile{}
	json.Unmarshal(response, &fileObject)

	return fileObject, nil
}

// DownloadFile : Downloads a file with 'ID' fileID.
func (box *Box) DownloadFile(fileID string, location string) error {
	box.RequestAccessToken()
	response, err := box.HTTPRequest("GET", "https://api.box.com/2.0/files/"+fileID+"/content", nil, nil)
	if err != nil {
		log.Println(err)
		return err
	}

	fInfo, err := box.GetFileInfo(fileID)
	file, err := os.Create(location + fInfo.Name)
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()

	_, err = file.Write(response)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// DeleteFile : Deletes a file in a specific folder with 'ID" fileID.
func (box *Box) DeleteFile(fileID string, etag string) error {
	box.RequestAccessToken()
	headers := make(map[string]string)
	headers["If-Match"] = etag
	_, err := box.HTTPRequest("DELETE", "https://api.box.com/2.0/files/"+fileID, nil, headers)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Folder Functions

// CreateFolder : Creates a new folder under the parent folder that has 'ID' parentFolderID.
func (box *Box) CreateFolder(name string, parentFolderID string) (*FolderObject, error) {
	box.RequestAccessToken()
	body := strings.NewReader(`{"name":"` + name + `", "parent": {"id": "` + parentFolderID + `"}}`)

	response, err := box.HTTPRequest("POST", "https://api.box.com/2.0/folders", body, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	folderObject := &FolderObject{}
	json.Unmarshal(response, &folderObject)

	return folderObject, nil
}

// GetFolderItems : Returns all the items contained inside the folder with 'ID' folderID.
func (box *Box) GetFolderItems(folderID string, limit int, offset int) (*ItemCollection, error) {
	box.RequestAccessToken()

	response, err := box.HTTPRequest("GET", "https://api.box.com/2.0/folders/"+folderID+"/items?limit="+strconv.Itoa(limit)+"&offset="+strconv.Itoa(offset), nil, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	items := &ItemCollection{}
	json.Unmarshal(response, &items)

	return items, nil
}

// DeleteFolder : Deletes the folder with 'ID' folderID.
func (box *Box) DeleteFolder(folderID string) error {
	box.RequestAccessToken()
	_, err := box.HTTPRequest("DELETE", "https://api.box.com/2.0/folders/"+folderID+"?recursive=true", nil, nil)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
