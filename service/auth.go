package pezauth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/oauth2"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	goauth2 "golang.org/x/oauth2"
)

//Constants to construct my oauth calls
const (
	ClientID       = "1083030294947-6g3bhhrgl3s7ul736jet625ajvp94f5p.apps.googleusercontent.com"
	ClientSecret   = "kfgM5mT3BqPQ84VeXsYokAK_"
	sessionName    = "pivotalpezauthservicesession"
	sessionSecret  = "shhh.donttellanyone"
	AuthFailStatus = 403
)

//Vars for my oauth calls
var (
	Scopes              = []string{"https://www.googleapis.com/auth/plus.me", "https://www.googleapis.com/auth/userinfo.email"}
	AuthFailureResponse = []byte(`{"error": "not logged in as a valid user, or the access token is expired"}`)
	allowedDomains      = []string{
		"pivotal.io",
	}
	OauthConfig *goauth2.Config
)

func isBlockedDomain(domain string) bool {
	isBlocked := true

	for _, d := range allowedDomains {

		if d == domain {
			isBlocked = false
		}
	}
	return isBlocked
}

func cleanVersionFromURI(uri string) string {
	var digitsRegexp = regexp.MustCompile(`-.*?\.`)
	match := digitsRegexp.FindStringSubmatch(uri)

	if len(match) > 0 {
		newS := strings.Replace(uri, match[0], ".", -1)
		uri = newS
	}

	if !strings.HasPrefix(uri, "http") {
		uri = fmt.Sprintf("https://%s", uri)
	}
	return uri
}

func getAppEnv() (appEnv *cfenv.App) {
	var err error

	if appEnv, err = cfenv.Current(); err != nil {
		panic(err)
	}
	return
}

func getAppURI() string {
	appEnv := getAppEnv()
	return cleanVersionFromURI(appEnv.ApplicationURIs[0])
}

//DomainChecker - check the authenticated users domain to see if it is in the whitelist
func DomainChecker(res http.ResponseWriter, tokens oauth2.Tokens) {
	userInfo := GetUserInfo(tokens)

	if domain, ok := userInfo["domain"]; !ok || tokens.Expired() || isBlockedDomain(domain.(string)) {
		res.WriteHeader(AuthFailStatus)
		res.Write(AuthFailureResponse)
	}
}

//DomainCheck - a handler to check if we are in a valid domain
var DomainCheck = func() martini.Handler {
	return DomainChecker
}()

//GetUserInfo - query googleapi for the authenticated users information
var GetUserInfo = func(tokens oauth2.Tokens) (userObject map[string]interface{}) {
	url := "https://www.googleapis.com/plus/v1/people/me"
	token := &goauth2.Token{
		AccessToken:  tokens.Access(),
		TokenType:    "Bearer",
		RefreshToken: tokens.Refresh(),
		Expiry:       tokens.ExpiryTime(),
	}
	client := OauthConfig.Client(goauth2.NoContext, token)
	resp, _ := client.Get(url)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &userObject)
	return
}

func setOauthConfig() {
	OauthConfig = &goauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Scopes:       Scopes,
		RedirectURL:  fmt.Sprintf("%s/oauth2callback", getAppURI()),
	}
}

type redisCreds interface {
	Pass() string
	Uri() string
}

//InitAuth - initializes authentication middleware for controllers
func InitSession(m *martini.ClassicMartini, rc redisCreds) {
	m.Use(render.Renderer())

	if rediStore, err := sessions.NewRediStore(10, "tcp", rc.Uri(), rc.Pass(), []byte(sessionSecret)); err == nil {
		m.Use(sessions.Sessions(sessionName, rediStore))
	}
}
