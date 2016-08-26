package pistis

import (
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	"errors"
	"time"
	"fmt"
	"github.com/Sirupsen/logrus"
	"sync"
	"github.com/cayleygraph/cayley/quad"
	"github.com/dgrijalva/jwt-go"
)

var (
	NoEmailError = errors.New("user email not found")
	NoPasswordError = errors.New("user password not found")
	UserExistedError = errors.New("user existed")
	UserNotExistedError = errors.New("user not existed")
	ContactNotExistedError = errors.New("contact not existed")
)

type User struct {
	*server
	Username string
	Password string
	Email    string
	Tel      string
	Avatar   string
	contacts map[string]*User
	clients  []string
	mu          sync.Mutex
}

func newUser(username, password, email, tel string) *User {
	var mu sync.Mutex
	return &User{
		Username :username,
		Password :password,
		Email    :email,
		Tel      :tel,
		clients  :make([]string,0),
		contacts :make(map[string]*User),
		mu:mu,
	}
}

func (u *User) ensureRunning() error {
	if u.server == nil {
		s, err := NewServer(fmt.Sprint("pistis/", u.Username, "/m"), mqtt_server)
		if err != nil {
			return err
		}
		s.mqttChannel.Subscribe(fmt.Sprint("pistis/", s.Name(), "/*/m"))
		u.server = s

		s.RegisterHandler("add-contact", func(s *server, m *Message) {
			contactName := m.Payload.(string)
			ok, err := u.addContact(contactName)
			if !ok {
				if err != nil && err == UserNotExistedError {
					s.send(&Message{
						TimeStamp :time.Now().Unix(),
						Type      :"add-contact-failed",
						Src       :s.Name(),
						Dst       :m.Src,
						Payload   :"user not existed",
					})
					return
				}

				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"add-contact-failed",
					Src       :s.Name(),
					Dst       :m.Src,
					Payload   :"failed to add contact",
				})
				return
			}

			s.send(&Message{
				TimeStamp :time.Now().Unix(),
				Type      :"add-contact-success",
				Src       :s.Name(),
				Dst       :m.Src,
				Payload   :"add contact success",
			})

		})

		s.RegisterHandler("remove-contact", func(s *server, m *Message) {
			contactName := m.Payload.(string)
			ok, err := u.removeContact(contactName)
			if !ok {
				if err != nil && err == ContactNotExistedError {
					s.send(&Message{
						TimeStamp :time.Now().Unix(),
						Type      :"remove-contact-failed",
						Src       :s.Name(),
						Dst       :m.Src,
						Payload   :"contact not existed",
					})
					return
				}

				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"remove-contact-failed",
					Src       :s.Name(),
					Dst       :m.Src,
					Payload   :"failed to remove contact",
				})
				return
			}

			s.send(&Message{
				TimeStamp :time.Now().Unix(),
				Type      :"remove-contact-success",
				Src       :s.Name(),
				Dst       :m.Src,
				Payload   :"remove contact failed",
			})
		})
	}
	s := u.server
	if s.IsRunning() == false {
		s.Start()
	}

	return nil
}

func (u *User) clientOpen(uuid string){
	u.addClient(uuid)

	userInfo := u.getUserInfo()
	u.send(&Message{
		TimeStamp :time.Now().Unix(),
		Type      :"opened",
		Src       :u.Name(),
		Dst       :u.clientTopic(uuid),
		Payload   :userInfo,
	})

	log.WithFields(logrus.Fields{
		"user name":u.Username,
		"name":u.Name(),
		"src":u.clientTopic(uuid),
	}).Debugln("opened")
}

func (u *User) clientClose(uuid string){
	u.removeClient(uuid)
}

func (u *User) clientTopic(uuid string) string{
	return fmt.Sprint("pistis/", u.Username, "/",uuid)
}
func (u *User)quads() []quad.Quad {
	return []quad.Quad{
		quad.Make(u.Username, "is", "User", ""),
		quad.Make(u.Username, "email", u.Email, ""),
		quad.Make(u.Username, "password", u.Password, ""),
		quad.Make(u.Username, "tel", u.Tel, ""),
		quad.Make(u.Username, "avatar", u.Avatar, ""),
	}
}

func checkUser(username, password string) bool {
	p := cayley.StartPath(store, quad.StringToValue(username)).Out("password")
	it := p.BuildIterator()
	defer it.Close()
	if it.Next() {
		pw := store.NameOf(it.Result())
		return password == pw.String()
	}
	return false
}

func checkUserEmail(email string) bool {
	p := cayley.StartPath(store, quad.StringToValue(email)).In("email")
	it := p.BuildIterator()
	defer it.Close()
	res:=false
	for it.Next() {
		res=true
		v:=store.NameOf(it.Result())
		log.WithFields(logrus.Fields{
			"res":v,
		}).Debugln("checkUserEmail")
	}
	return res
}

func checkUserName(username string) bool {
	p := cayley.StartPath(store, quad.StringToValue(username)).Has("is", quad.StringToValue("User"))
	it := p.BuildIterator()
	defer it.Close()
	res:=false
	for it.Next() {
		res=true
		v:=store.NameOf(it.Result())
		log.WithFields(logrus.Fields{
			"res":v,
		}).Debugln("checkUserName")
	}
	return res
}

func getUser(username string) (*User, error) {
	if checkUserName(username) == false {
		return nil, UserNotExistedError
	}

	user := newUser(username, "", "", "")

	getProperty:=func(name string)string{
		p := cayley.StartPath(store, quad.StringToValue(username)).Out(name)
		it:=p.BuildIterator()
		if it.Next() {
			v:=store.NameOf(it.Result()).Native()
			s,ok:=v.(string)
			if ok{
				return s
			}else{
				return ""
			}
		}else{
			return ""
		}
	}

	user.Email=getProperty("email")
	user.Password=getProperty("password")
	user.Tel=getProperty("tel")
	user.Avatar=getProperty("avatar")

	err := user.ensureRunning()

	return user, err
}

func (u *User) save() error {
	if checkUserName(u.Username) == false {
		for _,q:=range u.quads(){
			log.WithFields(logrus.Fields{
				"quad":q,
			}).Debugln("user save")
		}
		if err := store.AddQuadSet(u.quads()); err != nil {
			return err
		}
	} else {
		//update
	}

	u1,_:=getUser(u.Username)
	log.Debugln(u.Username,u1.Username)
	log.Debugln(u.Password,u1.Password)
	log.Debugln(u.Email,u1.Email)
	return nil
}

func (u *User) remove() error {
	tx := graph.NewTransaction()
	for _, quad := range u.quads() {
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}

func getUsername(email string) (string, error) {
	p := cayley.StartPath(store, quad.StringToValue(email)).In("email")
	it := p.BuildIterator()
	defer it.Close()
	if it.Next() {
		username := store.NameOf(it.Result()).String()
		return username, nil
	} else {
		return "", NoEmailError
	}
}

func (u *User) sendOnlineMsg() {
	for _, c := range u.contacts {
		for _, cl := range c.clients {
			log.Debugln(cl)
			//cl.sendUserOnlineMsg()
		}
	}
}

func (u *User) addContact(contact string) (bool, error) {
	if contact == u.Username || checkUserName(contact) == false {
		return false, UserNotExistedError
	}

	if err := store.AddQuad(cayley.Quad(u.Username, "contact", contact, "")); err != nil {
		return false, err
	}

	contactUser, err := getUser(contact)
	if err != nil {
		log.Println(err)
	} else {
		u.contacts[contact] = contactUser
	}

	return true, nil
}

func (u *User) removeContact(contact string) (bool, error) {
	if contact == u.Username || u.contacts[contact] == nil {
		return false, UserNotExistedError
	}

	if err := store.RemoveQuad(cayley.Quad(u.Username, "contact", contact, "")); err != nil {
		return false, err
	}

	u.contacts[contact] = nil

	return true, nil
}

type ContactMsg struct {
	Username string
	Email    string
	Tel      string
	Avatar   string
}

type UserInfo struct {
	Username string
	Email    string
	Tel      string
	Avatar   string
	Contacts []*ContactMsg
}

func (u *User) getUserInfo() *UserInfo {
	contacts := make([]*ContactMsg, 0)
	for _, contact := range u.contacts {
		contacts = append(contacts, &ContactMsg{
			Username    :contact.Username,
			Email        :  contact.Email,
			Tel        :  contact.Tel,
			Avatar      : contact.Avatar,
		})
	}
	userMsg := &UserInfo{
		Username:u.Username,
		Email:u.Email,
		Tel:u.Tel,
		Avatar:u.Avatar,
		Contacts:contacts,
	}
	return userMsg
}

func (u *User) hasClient(uuid string) bool{
	u.mu.Lock()
	defer u.mu.Unlock()
	for _,c:=range u.clients {
		if c==uuid {
			return true
		}
	}
	return false
}

func (u *User) addClient(uuid string) {
	if u.hasClient(uuid) {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.clients=append(u.clients,uuid)
}

func (u *User) removeClient(uuid string){
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.hasClient(uuid) {
		for i,c:=range u.clients {
			if c==uuid {
				u.clients= append(u.clients[:i], u.clients[i+1:]...)
				return
			}
		}
	}
}

func (u *User) token(uuid string) (*jwt.Token, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uuid": uuid,
		"username": u.Username,
		"password":u.Password,
	})
	return token, nil
}


