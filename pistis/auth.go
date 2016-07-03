package pistis

import (
	"net/http"
	"github.com/dgrijalva/jwt-go"
	"time"
	"github.com/go-martini/martini"
	"io/ioutil"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"fmt"
	"encoding/json"
	"errors"
)

var RSA_KEY = func() []byte {
	key, e := ioutil.ReadFile("RSA")
	if e != nil {
		panic(e.Error())
	}
	return key
}()

var RSA_PUB = func() []byte {
	key, e := ioutil.ReadFile("RSA.pub")
	if e != nil {
		panic(e.Error())
	}
	return key
}()

func float64Time(t time.Time) float64 {
	return float64(t.Unix())
}

type MyClaims struct {
	uuid     string
	username string
	password string
}

var (
	TokenNotForTheUserError = errors.New("token not for the user")
	TokenPasswordNotRightError = errors.New("token password not right")
)

func (c MyClaims) Valid() error {
	cl, err := getClient(c.uuid)
	if err != nil {
		return err
	}
	u, err := cl.user()
	if err != nil {
		return err
	}
	if u.Username != u.Username {
		return TokenNotForTheUserError
	}
	if u.Password != u.Password {
		return TokenPasswordNotRightError
	}
	return nil
}

func KeyFunc(token *jwt.Token) (interface{}, error) {
	err := token.Claims.Valid()
	return err == nil, err;
}

var Authenticator = func(c martini.Context, req *http.Request, res http.ResponseWriter) {
	token, _ := req.Cookie("Authorization")
	fmt.Println(token)
	fmt.Println(req.Header.Get("Authorization"))
	if req.RequestURI != "/auth" && req.RequestURI != "/register" {

		token:=req.Header.Get("token")

		if !checkToken(token) {
			res.WriteHeader(http.StatusForbidden)
			return
		}
	}
}

func checkToken(token string) bool {
	t, err := jwt.Parse(token, KeyFunc)

	if err != nil || !t.Valid {
		return false
	}

	claims:=t.Claims.(MyClaims)

	return checkUser(claims.username,claims.password) && checkClient(claims.uuid,claims.username)
}

/*rest server*/
type Auth struct {
	UUID     string
	Username string
	Email    string
	Password string
}

func AuthServer(r martini.Router) {
	r.Group("", func(r martini.Router) {
		r.Post("/auth", binding.Bind(Auth{}), auth)
		r.Post("/logout", logout)
		r.Post("/register", binding.Bind(Registration{}), register)
	})
}

func auth(auth Auth, r render.Render) {

	fmt.Println("auth")

	if auth.Username == "" || auth.Email == "" {
		r.Text(http.StatusBadRequest, "username and email can't both be empty")
		return
	}

	if auth.Password == "" {
		r.Text(http.StatusBadRequest, "password can't both be empty")
		return
	}

	username := auth.Username
	if username == "" {
		username = getUsername(auth.Email)
	}

	type AuthResult struct {
		Result  bool
		Message string
		Token   string
	}

	var result = func(result bool, message string, token *jwt.Token) {
		signedString, err := token.SignedString(RSA_KEY)
		if err != nil {
			r.Text(http.StatusInternalServerError, err.Error())
		}
		res := AuthResult{result, message, signedString}
		if jsonbyte, err := json.Marshal(res); err != nil {
			r.Text(http.StatusInternalServerError, err.Error())
		} else {
			if result {
				r.Text(http.StatusOK, string(jsonbyte))
			} else {
				r.Text(http.StatusConflict, string(jsonbyte))
			}
		}
	}

	u, err := getUser(username)
	if err != nil {
		result(false, err.Error(), nil)
		return
	}

	if auth.Password != u.Password {
		result(false, err.Error(), nil)
		return
	}

	c := &Client{UUID:auth.UUID, Username:auth.Username, user:u}
	err = c.save()
	if err != nil {
		r.Text(http.StatusInternalServerError, err.Error())
		return
	} else {
		result(false, err.Error(), nil)
		return
	}

	token, err := c.token()
	if err != nil {
		r.Text(http.StatusInternalServerError, err.Error())
		return
	}

	result(http.StatusOK, "ok", token)

	u.sendOnlineMsg()
}

func logout() {}

type Registration struct {
	UUID     string
	Email    string
	Password string
	Username string
}

func register(reg Registration, r render.Render) {

	type RegistrationResult struct {
		Result  bool
		Message string
		Token   string
	}

	var result = func(result bool, message string, token *jwt.Token) {
		signedString, err := token.SignedString(RSA_KEY)
		if err != nil {
			r.Text(http.StatusInternalServerError, err.Error())
		}
		res := RegistrationResult{result, message, signedString}
		if jsonbyte, err := json.Marshal(res); err != nil {
			r.Text(http.StatusInternalServerError, err.Error())
		} else {
			if result {
				r.Text(http.StatusOK, string(jsonbyte))
			} else {
				r.Text(http.StatusConflict, string(jsonbyte))
			}
		}
	}

	u := &User{Email:reg.Email, Password:reg.Password, Username:reg.Username, clients:make(map[string]*Client)}
	fmt.Println(reg)
	err := u.save()
	if err != nil {
		r.Text(http.StatusConflict, err.Error())
		return
	}

	err = u.ensureRunning()
	if err != nil {
		r.Text(http.StatusInternalServerError, err.Error())
		return
	}

	c := &Client{UUID:reg.UUID, Username:reg.Username, user:u}
	err = c.save()
	if err != nil {
		r.Text(http.StatusInternalServerError, err.Error())
		return
	}

	c.ensureRunning()

	token, err := c.token()
	if err != nil {
		r.Text(http.StatusInternalServerError, err.Error())
		return
	}

	result(http.StatusOK, "ok", token)
}




