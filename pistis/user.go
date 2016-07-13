package pistis

import (
	"github.com/google/cayley/quad"
	"github.com/google/cayley"
	g "github.com/google/cayley/graph"
	"errors"
	"encoding/json"
	"time"
	"fmt"
	"log"
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
	clients  map[string]*Client
}

func newUser(username, password, email, tel string) *User {
	return &User{
		Username :username,
		Password :password,
		Email    :email,
		Tel      :tel,
		clients  :make(map[string]*Client),
		contacts :make(map[string]*User),
	}
}

func (u *User) ensureRunning() error {
	if u.server == nil {
		s, err := NewServer(fmt.Sprint("pistis/", u.Username, "/m"), mqtt_server)
		if err != nil {
			return err
		}
		u.server = s

		type OpenAndCloseMsg struct {
			UUID     string
			Username string
			Token    string
		}
		parseOpenAndCloseMsg := func(m *Message) (*OpenAndCloseMsg, error) {
			jsonbyte := ([]byte)(m.Payload.(string))
			msg := &OpenAndCloseMsg{"", "", ""}
			err := json.Unmarshal(jsonbyte, m)
			return msg, err
		}

		type ContactsMsg struct {
			Username string
			Email    string
			Tel      string
			Avatar   string
		}

		type UserMsg struct {
			Username string
			Email    string
			Tel      string
			Avatar   string
			Contacts []*ContactsMsg
		}

		s.RegisterHandler("open", func(s *server, m *Message) {
			msg, err := parseOpenAndCloseMsg(m)
			if err != nil {
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :s.Name(),
					Dst       :m.Src,
					Payload   :"bad message",
				})
				return
			}

			if !checkTokenWithTokenInfo(msg.Token, msg.UUID, msg.Username) {
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :s.Name(),
					Dst       :m.Src,
					Payload   :"token is not valid",
				})
				return
			}

			c, _ := getClient(msg.UUID)
			if c == nil {
				c = &Client{UUID:msg.UUID, Username:msg.Username}
				c.ensureRunning()
			}

			contacts := make([]*ContactsMsg, 0)
			for _, contact := range u.contacts {
				contacts = append(contacts, *ContactsMsg{
					Username    :contact.Username,
					Email        :  contact.Email,
					Tel        :  contact.Tel,
					Avatar      : contact.Avatar,
				})
			}
			userMsg := &UserMsg{
				Username:u.Username,
				Email:u.Email,
				Tel:u.Tel,
				Avatar:u.Avatar,
				Contacts:contacts,
			}

			c.send(&Message{
				TimeStamp :time.Now().Unix(),
				Type      :"opened",
				Src       :s.Name(),
				Dst       :m.Src,
				Payload   :userMsg,
			})

		})

		s.RegisterHandler("close", func(s *server, m *Message) {
			msg, err := parseOpenAndCloseMsg(m)
			if err != nil {
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :s.Name(),
					Dst       :m.Src,
					Payload   :"bad message",
				})
				return
			}

			if !checkTokenWithTokenInfo(msg.Token, msg.UUID, msg.Username) {
				s.send(&Message{
					TimeStamp :time.Now().Unix(),
					Type      :"error",
					Src       :s.Name(),
					Dst       :m.Src,
					Payload   :"token is not valid",
				})
				return
			}

			c, _ := getClient(msg.UUID)
			if c != nil {
				c.Stop()
				u.clients[msg.UUID] = nil
			}
		})

		s.RegisterHandler("add-contact", func(s *server, m *Message) {
			contactName := m.Payload
			ok,err:=u.addContact(contactName)
			if !ok{
				if err!=nil && err==UserNotExistedError {
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
			contactName := m.Payload
			ok,err:=u.removeContact(contactName)
			if !ok {
				if err!=nil && err==ContactNotExistedError {
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

func (u *User)quads() []quad.Quad {
	return []quad.Quad{
		cayley.Quad(u.Username, "is", "User", ""),
		cayley.Quad(u.Username, "email", u.Email, ""),
		cayley.Quad(u.Username, "password", u.Password, ""),
		cayley.Quad(u.Username, "tel", u.Tel, ""),
		cayley.Quad(u.Username, "avatar", u.Avatar, ""),
	}
}

func checkUser(username, password string) bool {
	p := cayley.StartPath(store, username).Out("password")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		pw := store.NameOf(it.Result())
		return password == pw
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
		return nil, UserNotExistedError
	}

	p := cayley.StartPath(store, username).Out("password", "email", "tel", "avatar")
	it := p.BuildIterator()
	defer it.Close()

	user := newUser(username, "", "", "")
	user.Username = username
	if cayley.RawNext(it) {
		user.Password = store.NameOf(it.Result())
		if cayley.RawNext(it) {
			user.Avatar = store.NameOf(it.Result())
			if cayley.RawNext(it) {
				user.Email = store.NameOf(it.Result())
				if cayley.RawNext(it) {
					user.Tel = store.NameOf(it.Result())
				}
			}
		}
	}

	err := user.ensureRunning()

	return user, err
}

func (u *User) save() error {
	if checkUserName(u.Username) == false {
		if err := store.AddQuadSet(u.quads()); err != nil {
			return err
		}
	} else {
		//update
	}
	return nil
}

func (u *User) remove() error {
	tx := g.NewTransaction()
	for _, quad := range u.quads() {
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}

func getUsername(email string) (string, error) {
	p := cayley.StartPath(store, email).In("email")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		username := store.NameOf(it.Result())
		return username, nil
	} else {
		return "", NoEmailError
	}
}

func (u *User) sendOnlineMsg() {
	for _, c := range u.contacts {
		for _, cl := range c.clients {
			cl.sendUserOnlineMsg()
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
	if err!=nil{
		log.Println(err)
	}else{
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


