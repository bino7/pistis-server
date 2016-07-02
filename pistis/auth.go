package pistis

import (
	"net/http"
	"github.com/dgrijalva/jwt-go"
	"time"
	"github.com/go-martini/martini"
	"io/ioutil"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"pistis/mqtt"
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"strings"
	"github.com/golang/protobuf/proto"
	"sync"
	"fmt"
	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/martini-contrib/sessions"
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

func KeyFunc(token *jwt.Token) (interface{}, error) {
	exp, ok := token.Claims["exp"].(float64)
	if !ok {
		return jwt.ValidationErrorUnverifiable, jwt.ErrInvalidKey
	}
	if exp > float64Time(time.Now()) {
		return jwt.ValidationErrorExpired, jwt.ErrInvalidKey
	}

	email, ok1 := token.Claims["email"].(string)
	password, ok2 := token.Claims["password"].(string)
	if !ok1 || !ok2 {
		return jwt.ValidationErrorSignatureInvalid, jwt.ErrInvalidKey
	}
	if ok, _ := CheckUserPasswordByEmail(email, password); ok {
		token.Valid = true
		return RSA_PUB, nil
	}
	return nil, nil
}

var tokenDuration, _ = time.ParseDuration("720H")

func newToken(user *User) *jwt.Token {

	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = map[string]interface{}{
		"email":user.Email,
		"password":user.Password,
		"exp":float64Time(jwt.TimeFunc().Add(tokenDuration)),
	}
	return token
}

var Authenticator=func(c martini.Context, req *http.Request, res http.ResponseWriter) {
		token,_:=req.Cookie("Authorization")
		fmt.Println(token)
		fmt.Println(req.Header.Get("Authorization"))
		if req.RequestURI != "/auth" && req.RequestURI != "/register" {

			token, err := jwt.ParseFromRequest(req, KeyFunc)

			if err != nil || !token.Valid {
				res.WriteHeader(http.StatusForbidden)
				return
			}

			email, ok := token.Claims["email"].(string)
			if !ok {
				res.WriteHeader(http.StatusInternalServerError)
				return
			}

			if CheckUserEmail(email) == false {
				res.WriteHeader(http.StatusBadRequest)
				return
			}else {
				if username, err := GetUserNameByEmail(email); err != nil {
					res.WriteHeader(http.StatusInternalServerError)
				}else {
					req.Header.Add("username", username)
				}
			}

		}
	}

/*rest server*/
func AuthServer(r martini.Router) {
	r.Group("", func(r martini.Router) {
		r.Post("/auth", binding.Bind(User{}), auth)
		r.Post("/logout", logout)
		r.Post("/register", binding.Bind(registration{}), register)
		r.Post("/test",test)
	})
}

var servicesMutex sync.Mutex
var services = make(map[string]*mqtt.MqttService)
var serviceTimeout = "1800s"

type UserInfo struct {
	User  string
	Email string
	Token string
}

func auth(user User, r render.Render) {

	fmt.Println("auth")

	if user.Username=="" || user.Email=="" {
		r.Text(http.StatusBadRequest,"username and email can't both be empty")
		return
	}

	username:=user.Username
	if username == "" {
		username=GetUserName(user.Email)
	}
	token := createToken(&user)

	r.Header().Set("Authorization",fmt.Sprint("BEARER",token))

	if userInfo, err := getUserInfo(user.Username, user.Email); err != nil {
		r.Text(http.StatusInternalServerError, err.Error())
		return
	}else {
		userInfo.Token = token
		if b, err := json.Marshal(userInfo); err != nil {
			r.Text(http.StatusInternalServerError, err.Error())
			return
		}else {
			r.Text(http.StatusOK, string(b))
			return
		}
	}

}

/*func onMessage(user User, service *mqtt.MqttService) mqtt.MessageHandler {
	response := func(rsp proto.Message) {
		if m, err := proto.Marshal(rsp); err != nil {
			panic(strings.Join([]string{"marshal error", err.Error()}, ";"))
		}else {
			service.Send(rspTopic(user), m)
		}
	}
	return func(msg  MQTT.Message) {
		data := msg.Payload()
		rsp := new(message.Response)
		req := new(message.Request)
		if err := proto.UnmarshalMerge(data, req); err != nil {
			rsp.Result = false
			rsp.ErrorMessage = strings.Join([]string{"unmarshal error", err.Error()}, " ")
			response(rsp)
			return
		}

		timestamp := req.Timestamp
		rUser := req.User

		if user.Username != rUser {
			panic("user unmatch for mqtt msg")
		}

		rsp=&message.Response{
			Timestamp:timestamp,
			User:user.Username,
		}

		if req.GetUserInfoRequest() != nil {
			onUserInfoRequest(req.GetUserInfoRequest(),rsp, service)
		}

		if req.GetGetUserFriendsRequest() != nil {
			//rsp = onGetUserFriendsRequest(timestamp,user,req.GetUserFriendsRequest(),service)
		}
		if req.SearchRequest != nil {
		}

	}
}

func onTimeout(user User, service *mqtt.MqttService) mqtt.TimeoutHandler {
	return func() chan interface{} {
		done := make(chan interface{})
		go func() {
			servicesMutex.Lock()
			services[user.Username] = nil
			servicesMutex.Unlock()
			close(done)
		}()
		return done
	}
}

func reqTopic(user User) string {
	return strings.Join([]string{user.Username, "req"}, "/")
}
func rspTopic(user User) string {
	return strings.Join([]string{user.Username, "rsp"}, "/")
}
func notifyTopic(user User) string {
	return strings.Join([]string{user.Username, "notify"}, "/")
}

func onUserInfoRequest(req *message.UserInfoRequest,rsp *message.Response,	service *mqtt.MqttService) {
	if ui, err := getUserInfo(req.User, ""); err != nil {
		rsp.Result = false
		rsp.ErrorMessage = err.Error()
	}else {
		rsp.Result = true
		rsp.UserInfoResponse = &message.UserInfoResponse{
			User:ui.User,
			Email:ui.Email,
		}
	}
}

func getUserInfo(username, email string) (*UserInfo, error) {
	var f func(string) (*User, error)
	var p string

	if username != "" {
		f = GetUser
		p = username
	}else if email != "" {
		f = GetUserByEmail
		p = email
	}else {
		return nil, errors.New("username & email can not both be nil")
	}

	if user, err := f(p); err != nil {
		return nil, err
	}else {
		return &UserInfo{User:user.Username, Email:user.Email},nil
	}

}

func onGetFriendsRequest(m message.GetFriendsRequest) *message.GetFriendsResponse {
	return nil
}*/

func logout() {}

/*func test(r render.Render,req *http.Request,session sessions.Session){
	fmt.Println(r.Header())
	fmt.Println(req.Cookies())
	fmt.Println(session.Get("token"))
}*/

type registration struct {
	Email    string
	Password string
	Username string
}

func register(registration registration, r render.Render) {
	user := &User{Email:registration.Email, Password:registration.Password, Username:registration.Username}
	fmt.Println(registration)
	err := user.Save()
	if err != nil {
		r.Text(http.StatusConflict, err.Error())
		return
	}
	token := createToken(user)
	if ui, err := getUserInfo(user.Username, user.Email); err != nil {
		r.Text(http.StatusInternalServerError, err.Error())
		return
	}else {
		ui.Token = token
		if b, err := json.Marshal(ui); err != nil {
			r.Text(http.StatusInternalServerError, err.Error())
			return
		}else {
			r.Text(http.StatusOK, string(b))
			return
		}
	}
}

func createToken(user *User) string {
	token := newToken(user)
	signedString, err := token.SignedString(RSA_KEY)
	if err != nil {
		panic(err)
	}
	return signedString
}


