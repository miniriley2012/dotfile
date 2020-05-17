package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/knoebber/dotfile/db"
	"github.com/knoebber/dotfile/usererr"
)

const minPassLength = 8

// Redirects to index when session exists.
func checkSession(w http.ResponseWriter, r *http.Request, p *Page) (done bool) {
	if p.Session != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return true
	}
	return false
}

func createHandleLogin(secure bool) pageBuilder {
	return func(w http.ResponseWriter, r *http.Request, p *Page) (done bool) {
		return login(w, r, p, secure)
	}
}

func login(w http.ResponseWriter, r *http.Request, p *Page, secure bool) (done bool) {
	s, err := db.UserLogin(r.Form.Get("username"), r.Form.Get("password"))
	if err != nil {
		// Print the real error and show the user a generic catch all.
		log.Print(err)
		return p.setError(w, usererr.Invalid("Username or password is incorrect."))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.Session,
		Secure:   secure,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return true
}

func createHandleSignup(secure bool) pageBuilder {
	return func(w http.ResponseWriter, r *http.Request, p *Page) (done bool) {
		username := r.Form.Get("username")
		password := r.Form.Get("password")
		confirm := r.Form.Get("confirm")

		if len(password) < minPassLength {
			msg := fmt.Sprintf("Password must be at least %d characters.", minPassLength)
			return p.setError(w, usererr.Invalid(msg))
		}

		if password != confirm {
			p.setError(w, usererr.Invalid("Passwords do not match."))
			return false
		}

		_, err := db.CreateUser(username, password, nil)
		if err != nil {
			return p.setError(w, err)
		}

		return login(w, r, p, secure)
	}
}

func handleEmail(w http.ResponseWriter, r *http.Request, p *Page) (done bool) {
	if err := db.UpdateEmail(p.Session.UserID, r.Form.Get("email")); err != nil {
		return p.setError(w, err)
	}
	p.Data["email"] = r.Form.Get("email")

	p.flashSuccess("Updated email")
	return false
}

func handlePassword(w http.ResponseWriter, r *http.Request, p *Page) (done bool) {
	currentPass := r.Form.Get("current")

	newPass := r.Form.Get("new")
	confirm := r.Form.Get("confirm")

	if len(newPass) < minPassLength {
		return p.setError(w, usererr.Invalid(fmt.Sprintf("Password must be %d or more characters.", minPassLength)))
	}

	if newPass != confirm {
		return p.setError(w, usererr.Invalid("Confirm does not match."))
	}

	if err := db.UpdatePassword(p.Session.UserID, currentPass, newPass); err != nil {
		return p.setError(w, err)
	}

	p.flashSuccess("Updated password")
	return false
}

func loadUser(w http.ResponseWriter, r *http.Request, p *Page) (done bool) {
	var (
		user  *db.User
		email string
		err   error
	)

	if p.Vars["username"] == "" {
		user, err = db.GetUser(p.Session.UserID, "")
	} else {
		user, err = db.GetUser(0, p.Vars["username"])
	}

	if err != nil {
		return p.setError(w, err)
	}

	if user.Email != nil {
		email = *user.Email
	}

	p.Title = user.Username

	p.Data["email"] = email
	p.Data["joined"] = user.JoinDate()
	return
}

func handleLogout(w http.ResponseWriter, r *http.Request, p *Page) (done bool) {
	if err := db.Logout(p.Session.ID); err != nil {
		log.Print(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookie,
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return true
}

func getLoginHandler(secure bool) http.HandlerFunc {
	return createHandler(&pageDescription{
		templateName: "login.tmpl",
		title:        loginTitle,
		loadData:     checkSession,
		handleForm:   createHandleLogin(secure),
	})
}

func getSignupHandler(secure bool) http.HandlerFunc {
	return createHandler(&pageDescription{
		templateName: "signup.tmpl",
		title:        signupTitle,
		loadData:     checkSession,
		handleForm:   createHandleSignup(secure),
	})
}

func getUserHandler() http.HandlerFunc {
	return createHandler(&pageDescription{
		templateName: "user.tmpl",
		title:        "<username>",
		loadData:     loadUser,
	})
}

func getEmailHandler() http.HandlerFunc {
	return createHandler(&pageDescription{
		templateName: "email.tmpl",
		title:        emailTitle,
		loadData:     loadUser,
		handleForm:   handleEmail,
		protected:    true,
	})
}

func getPasswordHandler() http.HandlerFunc {
	return createHandler(&pageDescription{
		templateName: "password.tmpl",
		title:        passwordTitle,
		handleForm:   handlePassword,
		protected:    true,
	})
}

func getLogoutHandler() http.HandlerFunc {
	return createHandler(&pageDescription{
		handleForm: handleLogout,
		protected:  true,
	})
}
