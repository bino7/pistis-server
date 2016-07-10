package pistis

import (
	"github.com/google/cayley/quad"
	"github.com/google/cayley"
	g "github.com/google/cayley/graph"
	"errors"
	"encoding/json"
	"time"
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

func newUser(username,password,email,tel string) *User{
	return &User{
		Username :username,
		Password :password,
		Email    :email,
		Tel      :tel,
		clients  :make(map[string] *Client),
		contacts :make(map[string] *User),
	}
}

func (u *User) ensureRunning() error{
	if u.server==nil {
		s,err:=NewServer(u.Username,mqtt_server)
		if err!=nil {
			return err
		}
		u.server=s

		type ClientMsg struct {
			UUID			string
			Username 	string
			Token 	 	string
		}
		parseClientMsg:=func(m *Message) (*ClientMsg,error){
			jsonbyte:=([]byte)(m.Payload.(string))
			msg:=&ClientMsg{"","",""}
			err:=json.Unmarshal(jsonbyte,m)
			return msg,err
		}

		s.RegisterHandler("open",func(s *server, m *Message) {
			msg,err:=parseClientMsg(m)
			if err!=nil{
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :"pistis",
					Dst       :m.Src,
					Payload   :err.Error(),
				})
				return
			}

			if !checkToken(msg.Token) {
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :"pistis",
					Dst       :m.Src,
					Payload   :"token is not valid",
				})
				return
			}

			c,_:=getClient(msg.UUID)
			if c==nil {
				c=&Client{UUID:msg.UUID,Username:msg.Username}
				c.ensureRunning()
			}

		})

		s.RegisterHandler("close",func(s *server,m *Message){
			msg,err:=parseClientMsg(m)
			if err!=nil{
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :"pistis",
					Dst       :m.Src,
					Payload   :err.Error(),
				})
				return
			}

			if !checkToken(msg.Token) {
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :"pistis",
					Dst       :m.Src,
					Payload   :"token is not valid",
				})
				return
			}

			c,_:=getClient(msg.UUID)
			if c!=nil {
				c.Stop()
				u.clients[msg.UUID]=nil
			}
		})

		s.RegisterHandler("add-contact",func(s *server,m *Message){
			
		})

		s.RegisterHandler("remove-contact",func(s *server,m *Message){

		})
	}
	s := u.server
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

func checkUserEmail(email string) bool {
	p := cayley.StartPath(store, email).In("email")
	it := p.BuildIterator()
	defer it.Close()
	return cayley.RawNext(it)
}

func checkUserName(username string) bool {
	p := cayley.StartPath(store, username).Has("is", "User")
	it := p.BuildIterator()
	defer it.Close()
	return cayley.RawNext(it)
}

func getUser(username string) (*User, error) {
	if checkUserName(username) == false {
		return nil, NoUserError
	}

	p := cayley.StartPath(store, username).Out("password","email","tel")
	it := p.BuildIterator()
	defer it.Close()

	user := newUser(username,"","","")
	user.Username = username
	if cayley.RawNext(it) {
		user.Password = store.NameOf(it.Result())
		if cayley.RawNext(it) {
			user.Email = store.NameOf(it.Result())
			if cayley.RawNext(it) {
				user.Tel = store.NameOf(it.Result())
			}
		}
	}

	err := user.ensureRunning()

	return user, err
}

func (u *User) save() error {
	if checkUserName(u.Username) == false {
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
	for _,c:= range u.contacts {
		for _,cl:= range c.clients{
			cl.sendUserOnlineMsg()
		}
	}
}


