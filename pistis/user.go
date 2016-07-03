package pistis

import (
	"github.com/google/cayley/quad"
	"github.com/google/cayley"
	g "github.com/google/cayley/graph"
	"errors"
	"encoding/json"
	"time"
	"github.com/dgrijalva/jwt-go"
)

var (
	NoUserError = errors.New("user not found")
	NoEmailError = errors.New("user email not found")
	NoPasswordError = errors.New("user password not found")
	UserExistedError = errors.New("user existed")
)

type User struct {
	*server
	Username string
	Password string
	Email    string
	Tel      string
	clients  map[string] *Client
	contacts map[string] *User
}

func (u *User) ensureRunning() error{
	s:=u.server
	if s==nil {
		s,err:=NewServer(u.Username,mqtt_server)
		if err!=nil {
			return err
		}
		u.server=s

		s.RegisterHandler("client-opened",func(s *server,m *Message){
			c:=m.Payload.(*Client)
			u.clients[c.UUID]=c
		})
		s.RegisterHandler("client-closed",func(s *server,m *Message){
			c:=m.Payload.(*Client)
			u.clients[c.UUID]=nil
		})
		s.RegisterHandler("online",func(s *server,m *Message){

		})
		s.RegisterHandler("open",func(s *server, m *Message) {
			type OpenMsg struct {
				Username 	string
				Token 	 	string
			}
			jsonbyte:=([]byte)(m.Payload.(string))
			openMsg:=OpenMsg{"",""}
			if err:=json.Unmarshal(jsonbyte,m);err!=nil{
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :"pistis",
					Dst       :m.Src,
					Payload   :err.Error(),
				})
			}

			token,_:=jwt.Parse(openMsg.Token,KeyFunc)
			if !token.Valid {
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :"pistis",
					Dst       :m.Src,
					Payload   :"token is not valid",
				})
			}
		})
	}

	if s.IsRunning()==false {
		s.Start()
	}

	return nil
}

func (u *User)quads() []quad.Quad {
	return []quad.Quad{
		cayley.Quad(u.Username, "is", "User", ""),
		cayley.Quad(u.Username, "email", u.Email, ""),
		cayley.Quad(u.Username, "password", u.Password, ""),
		cayley.Quad(u.Username, "tel", u.Tel, ""), }
}

func checkUser(username,password string) bool {
	p := cayley.StartPath(store, username).Out("password")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		pw:=store.NameOf(it.Result())
		return password==pw
	}
	return false
}

func CheckUserEmail(email string) bool {
	p := cayley.StartPath(store, email).In("email")
	it := p.BuildIterator()
	defer it.Close()
	return cayley.RawNext(it)
}

func CheckUserName(username string) bool {
	p := cayley.StartPath(store, username).Has("is", "User")
	it := p.BuildIterator()
	defer it.Close()
	return cayley.RawNext(it)
}

func getUser(username string) (*User, error) {
	if CheckUserName(username) == false {
		return nil, NoUserError
	}

	p := cayley.StartPath(store, username).Out("email", "password","tel")
	it := p.BuildIterator()
	defer it.Close()

	user := &User{clients:make(map[string]*Client)}
	user.Username = username
	if cayley.RawNext(it) {
		user.Email = store.NameOf(it.Result())
		if cayley.RawNext(it) {
			user.Password = store.NameOf(it.Result())
			if cayley.RawNext(it) {
				user.Tel = store.NameOf(it.Result())
			}
		}
	}

	err := user.ensureRunning()

	return user, err
}

func (u *User) save() error {
	if CheckUserName(u.Username) == false {
		if err:=store.AddQuadSet(u.quads());err!=nil {
			return err
		}
	}else{
		//update
	}
	return nil
}

func (u *User) remove()error{
	tx:=g.NewTransaction()
	for _,quad:=range u.quads(){
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}

func getUsername(email string) (string, error) {
	p := cayley.StartPath(store, email).In("email")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		username:=store.NameOf(it.Result())
		return username,nil
	}else{
		return "",NoEmailError
	}
}

func (u *User) sendOnlineMsg() {
	for _,c:= range u.clients {
		c.sendUserOnlineMsg()
	}
}


