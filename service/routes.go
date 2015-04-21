package pezauth

import (
	"fmt"
	"log"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/oauth2"
	"github.com/martini-contrib/render"
)

//Constants to construct routes with
const (
	UserParam   = "user"
	APIVersion1 = "v1"
	AuthGroup   = "auth"
	APIKeys     = "/api-keys"
	StaticPath  = "public"
)

//formatted strings based on constants, to be used in URLs
var (
	APIKey        = fmt.Sprintf("/api-key/:%s", UserParam)
	URLAuthBaseV1 = fmt.Sprintf("/%s/%s", APIVersion1, AuthGroup)
)

//Response - generic response object
type Response struct {
	User     map[string]interface{}
	APIKey   string
	ErrorMsg string
}

//InitRoutes - initialize the mappings for controllers against valid routes
func InitRoutes(m *martini.ClassicMartini, redisConn Doer) {
	setOauthConfig()
	m.Use(render.Renderer())
	m.Use(martini.Static(StaticPath))
	authKey := NewAuthKeyV1(NewKeyGen(redisConn, &GUIDMake{}))
	m.Get("/info", authKey.Get())
	m.Get("/me", oauth2.Google(OauthConfig), oauth2.LoginRequired, DomainCheck, NewMeController().Get())

	m.Get("/", oauth2.Google(OauthConfig), oauth2.LoginRequired, DomainCheck, func(params martini.Params, log *log.Logger, r render.Render, tokens oauth2.Tokens) {
		userInfo := GetUserInfo(tokens)
		r.HTML(200, "index", userInfo)
	})

	m.Group(URLAuthBaseV1, func(r martini.Router) {
		r.Put(APIKey, authKey.Put())       //this will re-generate a new key for the user or create one if it doesnt exist
		r.Post(APIKey, authKey.Post())     //this will generate a key for the user or do nothing
		r.Get(APIKey, authKey.Get())       //will return the key for the username (pivotal.io email) it is given... this needs to be locked in that only the current user or admin will receive a result
		r.Delete(APIKey, authKey.Delete()) //this will remove the key from the user

		r.Get(APIKeys, authKey.Get()) //will return the list of current keys and its associated user
	}, oauth2.Google(OauthConfig), oauth2.LoginRequired, DomainCheck)
}
