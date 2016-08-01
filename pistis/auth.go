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
	"log"
)

var RSA_KEY = func() []byte {
	key, e := ioutil.ReadFile("RSA.key")
	if e != nil {
		panic(e.Error())
	}
	return key
}()

var RSA_PUB = func() []byte {
	key, e := ioutil.ReadFile("RSA.public.key")
	if e != nil {
		panic(e.Error())
	}
	return key
}()

func float64Time(t time.Time) float64 {
	return float64(t.Unix())
}

type MyClaims struct {
	UUID     string
	Username string
	Password string
}

var (
	TokenNotForTheUserError = errors.New("token not for the user")
	TokenPasswordNotRightError = errors.New("token password not right")
)

func (c MyClaims) Valid() error {
	if checkUser(c.Username, c.Password) && checkClient(c.Username, c.UUID) {
		return nil
	}
	return jwt.ErrInvalidKey
}

func KeyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
	}
	if _, ok := token.Claims.(jwt.MapClaims); !ok {
		return nil, nil
	} /*else if !checkUser(claims["username"].(string), claims["password"].(string)) || !checkClient(claims["username"].(string), claims["uuid"].(string)) {
		return nil, nil
	}*/
		return RSA_KEY, nil
}

var Authenticator = func(c martini.Context, req *http.Request, res http.ResponseWriter) {
	token, _ := req.Cookie("Authorization")
	fmt.Println(token)
	if req.RequestURI != "/auth" && req.RequestURI != "/register" {

		token := req.Header.Get("token")

		if !checkToken(token) {
			res.WriteHeader(http.StatusForbidden)
			return
		}
	}
}

func checkToken(tokenstr string) bool {
	token, err := jwt.Parse(tokenstr, KeyFunc)
	return err == nil && token.Valid
}
func checkTokenWithUUID(tokenstr, uuid string) bool {
	token, err := jwt.Parse(tokenstr, KeyFunc)
	if err != nil || !token.Valid {
		return false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims["uuid"] == uuid
	} else {
		return false
	}
}

func checkTokenWithTokenInfo(tokenstr,username, uuid string) bool {
	token, err := jwt.Parse(tokenstr, KeyFunc)
	if err != nil || !token.Valid {
		return false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims["uuid"] == uuid && claims["username"] == username
	} else {
		return false
	}
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

func auth(auth Auth, r render.Render, req *http.Request) {

	type AuthResult struct {
		Result  bool
		Message string
		Token   string
	}

	var result = func(status int, result bool, message string, token *jwt.Token) {
		signedString := ""
		if token != nil {
			s, err := token.SignedString(RSA_KEY)

			if err != nil {
				panic(err.Error())
			} else {
				signedString = s
			}
		}

		res := AuthResult{result, message, signedString}

		if jsonbyte, err := json.Marshal(res); err != nil {
			panic(err.Error())
		} else {
			r.Text(status, string(jsonbyte))
		}
	}

	if auth.Username == "" && auth.Email == "" {
		result(http.StatusBadRequest, false, "username and email can't both be empty", nil)
		return
	}

	if auth.Password == "" {
		log.Println(http.StatusBadRequest, "password can't be empty")
		result(http.StatusBadRequest, false, "password can't be empty", nil)
		return
	}

	username := auth.Username
	if username == "" {
		username, _ = getUsername(auth.Email)
	}

	u, err := getUser(username)
	if err != nil {
		result(http.StatusConflict, false, "can not find user", nil)
		return
	}

	if auth.Password != u.Password {
		log.Println(auth.Password, u.Password)
		result(http.StatusConflict, false, "password not matched", nil)
		return
	}

	c := &Client{UUID:auth.UUID, Username:auth.Username, user:u}
	err = c.save()
	if err != nil {
		result(http.StatusInternalServerError, false, "failed to start client", nil)
		return
	}

	token, err := c.token()
	if err != nil {
		result(http.StatusInternalServerError, false, "failed to create token", nil)
		return
	}

	result(http.StatusOK, true, "ok", token)

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
	log.Println(reg)
	type RegistrationResult struct {
		Result  bool
		Message string
		Token   string
	}

	var result = func(status int, result bool, message string, token *jwt.Token) {
		signedString := ""
		if token != nil {
			s, err := token.SignedString(RSA_KEY)

			if err != nil {
				panic(err.Error())
			} else {
				signedString = s
			}
		}

		res := RegistrationResult{result, message, signedString}
		if jsonbyte, err := json.Marshal(res); err != nil {
			panic(err.Error())
		} else {
			r.Text(status, string(jsonbyte))
		}
	}

	u := &User{Email:reg.Email,
		Password:reg.Password,
		Username:reg.Username,
		clients:make(map[string]*Client),
		contacts:make(map[string]*User)}

	if checkUserName(u.Username) || checkUserEmail(u.Email) {
		result(http.StatusNotAcceptable, false, "username or email exited", nil)
		return
	}

	err := u.save()
	if err != nil {
		result(http.StatusInternalServerError, false, "", nil)
		return
	}

	err = u.ensureRunning()
	if err != nil {
		result(http.StatusInternalServerError, false, "", nil)
		return
	}

	c := &Client{UUID:reg.UUID, Username:reg.Username, user:u}
	err = c.save()
	if err != nil {
		result(http.StatusInternalServerError, false, "", nil)
		return
	}

	c.ensureRunning()

	token, err := c.token()
	if err != nil {
		result(http.StatusInternalServerError, false, "create token error", nil)
		return
	}

	result(http.StatusCreated, true, "", token)
}




