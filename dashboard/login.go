package dashboard

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/conference"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func loginConfig(config *conference.Configuration, protocol, host string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.LoginClient.ID,
		ClientSecret: config.LoginClient.Secret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		RedirectURL:  fmt.Sprintf("%s://%s/login/callback", protocol, host),
		Endpoint:     google.Endpoint,
	}
}

func (s *service) Serve_dashboard_login(rc *requestContext) error {
	p := make([]byte, 16)
	rand.Read(p)
	state := fmt.Sprintf("%x", p)
	http.SetCookie(rc.Response, &http.Cookie{
		Name:     "state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.Protocol == "https",
		SameSite: http.SameSiteLaxMode,
	})
	config := rc.Conference.Configuration
	http.Redirect(rc.Response, rc.Request, loginConfig(config, s.Protocol, rc.Request.Host).AuthCodeURL(state), http.StatusSeeOther)
	return nil
}

func (s *service) Serve_login_callback(rc *requestContext) error {
	cookie, err := rc.Request.Cookie("state")
	if err != nil || cookie.Value != rc.FormValue("state") {
		return &application.HTTPError{Status: http.StatusForbidden, Err: errors.New("bad state value")}
	}
	config := rc.Conference.Configuration
	lconfig := loginConfig(config, s.Protocol, rc.Request.Host)

	token, err := lconfig.Exchange(rc.Ctx, rc.FormValue("code"))
	if err != nil {
		return &application.HTTPError{Status: http.StatusBadRequest, Err: err}
	}
	client := lconfig.Client(rc.Ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var userInfo struct {
		Email string
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return err
	}

	id := strings.ToLower(userInfo.Email)
	if rc.Conference.IsStaff(id) {
		rc.setStaffID(s.Protocol, id)
		rc.Logf("login success: %s", id)
	} else {
		rc.SetFlashMessage("info", fmt.Sprintf("The account %s is not authorized to access this application.", id))
		rc.Logf("login fail: %s", id)
	}
	http.Redirect(rc.Response, rc.Request, "/dashboard", http.StatusSeeOther)
	return nil
}

func (s *service) Serve_dashboard_logout(rc *requestContext) error {
	rc.setStaffID(s.Protocol, "")
	http.Redirect(rc.Response, rc.Request, "/dashboard", http.StatusSeeOther)
	return nil
}

func (rc *requestContext) setStaffID(protocol string, id string) {
	if id == "" {
		http.SetCookie(rc.Response, &http.Cookie{
			Name:     "staff",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			HttpOnly: true,
			Secure:   protocol == "https",
			SameSite: http.SameSiteStrictMode,
		})
		return
	}

	config := rc.Conference.Configuration
	const maxAge = 30 * 24 * 3600
	http.SetCookie(rc.Response, &http.Cookie{
		Name: "staff",
		Value: application.SignValue(
			config.CookieKey,
			maxAge,
			application.EncodeStringsForCookie(id)),
		MaxAge:   maxAge,
		Path:     "/",
		HttpOnly: true,
		Secure:   protocol == "https",
		SameSite: http.SameSiteStrictMode,
	})
}
