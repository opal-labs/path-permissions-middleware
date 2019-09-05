package filepermissions

import (
	"log"
	"net/http"
	"strings"
)

//Error is an error generated by this package
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

//AccessLevel represents an access level
type AccessLevel string

//The allowed access levels
const (
	Read      AccessLevel = "R"
	ReadWrite AccessLevel = "RW"
)

//PathGrant represents permissions granted to access the Path with the AccessLevel
type PathGrant struct {
	Access AccessLevel
	Path   string
}

//Helpers is the interface that wraps the loading of grants for a user based on the provided http.Request
type Helpers interface {
	GetUserGrants(r *http.Request) ([]PathGrant, error)
	GetRequestedPath(r *http.Request) (string, error)
}

//CreateFilePermissionsMiddleware creates a new middleware
func CreateFilePermissionsMiddleware(helpers Helpers) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//Load the grants for the request
			grants, err := helpers.GetUserGrants(r)
			if err != nil {
				handleError(w, err)
				return
			}
			//Get the requested path
			path, err := helpers.GetRequestedPath(r)
			if err != nil {
				handleError(w, err)
				return
			}
			//Make sure the requested path is allowed by the list of grants
			allowed := false
			for _, grant := range grants {
				if strings.HasPrefix(path, grant.Path) {
					if r.Method == http.MethodGet && (grant.Access == Read || grant.Access == ReadWrite) {
						allowed = true
						break
					} else if (r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut || r.Method == http.MethodDelete) && grant.Access == ReadWrite {
						allowed = true
						break
					}
				}
			}

			//If the requested path is not allowed by the grants, return unauthorized
			if allowed == false {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("401 - Unauthorized"))
				return
			}

			next.ServeHTTP(w, r) //Next middleware
		})
	}
}

//Handle any errors returned by the helpers
func handleError(w http.ResponseWriter, e error) {
	err, ok := e.(*Error)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	} else {
		w.WriteHeader(err.Code)
		if err.Code == http.StatusUnauthorized {
			w.Write([]byte("401 - Unauthorized"))
		} else if err.Code == http.StatusBadRequest {
			w.Write([]byte("400 - Bad Request"))
		}
	}
	log.Println("error in file permissions middleware:", err, e)
}
