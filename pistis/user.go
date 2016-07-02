package pistis

import (
	"github.com/google/cayley/quad"
	"github.com/google/cayley"
	g "github.com/google/cayley/graph"
	"errors"
)

var (
	NoUserError = errors.New("user not found")
	NoEmailError = errors.New("user email not found")
	NoPasswordError = errors.New("user password not found")
	UserExistedError = errors.New("user existed")
)

type User struct {
	Username string
	Password string
	Email    string
	Tel      string
}

func (u *User)quads() []quad.Quad {
	return []quad.Quad{
		cayley.Quad(u.Username, "is", "User", ""),
		cayley.Quad(u.Username, "email", u.Email, ""),
		cayley.Quad(u.Username, "password", u.Password, ""),
		cayley.Quad(u.Username, "tel", u.Tel, ""), }
}

func CheckUserEmail(email string) bool {
	p := cayley.StartPath(store, email).In("email")
	it := p.BuildIterator()
	defer it.Close()
	return cayley.RawNext(it)
}

func CheckUserName(name string) bool {
	p := cayley.StartPath(store, name).Has("is", "User")
	it := p.BuildIterator()
	defer it.Close()
	return cayley.RawNext(it)
}

func GetUser(username string) (*User, error) {
	if CheckUserName(username) == false {
		return nil, NoUserError
	}

	p := cayley.StartPath(store, username).Out("email", "password","tel")
	it := p.BuildIterator()
	defer it.Close()

	user := &User{}
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

	return user, nil
}

func (u *User) Save() error {
	if CheckUserName(u.Username) == false {
		return store.AddQuadSet(u.quads())
	}else{
		//update
	}

	return nil
}

func (u *User) Remove()error{
	tx:=g.NewTransaction()
	for _,quad:=range u.quads(){
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}

func GetUserName(email string) (string, error) {
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
